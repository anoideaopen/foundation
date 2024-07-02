/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: [Default license](LICENSE)
*/

// Package stub mocked provides APIs for the chaincode to access its state
// variables, transaction context and call other chaincodes.
package stub

import (
	"container/list"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go/msp"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/op/go-logging"
)

const module = "mock"

func init() {
	time.Local = time.UTC
	logging.SetLevel(logging.ERROR, module)
}

// ErrFuncNotImplemented is returned when a function is not implemented
const ErrFuncNotImplemented = "function %s is not implemented"

// Stub is an implementation of ChaincodeStubInterface for unit testing chaincode.
// Use this instead of ChaincodeStub in your chaincode's unit test calls to Init or Invoke.
type Stub struct {
	// A pointer back to the chaincode that will invoke this, set by constructor.
	// If a peer calls this stub, the chaincode will be invoked from here.
	cc                     shim.Chaincode
	Args                   [][]byte          // arguments the stub was called with
	Name                   string            // A nice name that can be used for logging
	State                  map[string][]byte // State keeps name value pairs
	Keys                   *list.List        // Keys stores the list of mapped values in lexical order registered list of other Stub chaincodes that can be called from this Stub
	Invokables             map[string]*Stub
	TxID                   string // stores a transaction uuid while being Invoked / Deployed
	TxTimestamp            *timestamp.Timestamp
	signedProposal         *pb.SignedProposal // mocked signedProposal
	ChannelID              string             // stores a channel ID of the proposal
	PvtState               map[string]map[string][]byte
	EndorsementPolicies    map[string]map[string][]byte // stores per-key endorsement policy, first map index is the collection, second map index is the key
	ChaincodeEventsChannel chan *pb.ChaincodeEvent      // channel to store ChaincodeEvents
	Decorations            map[string][]byte
	creator                []byte
	logger                 *logging.Logger
	transientMap           map[string][]byte
}

// NewMockStub - Constructor to config the internal State map
func NewMockStub(name string, cc shim.Chaincode) *Stub {
	s := new(Stub)
	s.Name = name
	s.cc = cc
	s.State = make(map[string][]byte)
	s.PvtState = make(map[string]map[string][]byte)
	s.EndorsementPolicies = make(map[string]map[string][]byte)
	s.Invokables = make(map[string]*Stub)
	s.Keys = list.New()
	s.ChaincodeEventsChannel = make(chan *pb.ChaincodeEvent, 100) //nolint:gomnd    // define large capacity for non-blocking setEvent calls.
	s.Decorations = make(map[string][]byte)
	s.logger = logging.MustGetLogger("mock")
	s.transientMap = make(map[string][]byte)

	return s
}

// GetTxID returns the transaction ID for the current chaincode invocation request.
func (stub *Stub) GetTxID() string {
	return stub.TxID
}

// GetChannelID returns the channel ID for the proposal for the current chaincode invocation request.
func (stub *Stub) GetChannelID() string {
	return stub.ChannelID
}

// GetArgs returns the arguments for the chaincode invocation request.
func (stub *Stub) GetArgs() [][]byte {
	return stub.Args
}

// GetStringArgs returns the arguments for the chaincode invocation request as strings.
func (stub *Stub) GetStringArgs() []string {
	args := stub.GetArgs()
	strargs := make([]string, 0, len(args))
	for _, barg := range args {
		strargs = append(strargs, string(barg))
	}
	return strargs
}

// GetFunctionAndParameters returns the first argument as the function name and the rest of the arguments as parameters in a string array.
func (stub *Stub) GetFunctionAndParameters() (function string, params []string) {
	allArgs := stub.GetStringArgs()
	function = ""
	params = []string{}
	if len(allArgs) >= 1 {
		function = allArgs[0]
		params = allArgs[1:]
	}
	return
}

// MockTransactionStart is used to indicate to a chaincode that it is part of a transaction.
// This is important when chaincodes invoke each other.
// Stub doesn't support concurrent transactions at present.
func (stub *Stub) MockTransactionStart(txID string) {
	stub.TxID = txID
	stub.setSignedProposal(&pb.SignedProposal{})
	stub.setTxTimestamp(createUtcTimestamp())
}

// MockTransactionEnd ends a mocked transaction, clearing the UUID.
func (stub *Stub) MockTransactionEnd(_ string) { // uuid
	stub.signedProposal = nil
	stub.TxID = ""
}

// MockPeerChaincode registers a peer chaincode with this Stub
// invokeableChaincodeName is the name or hash of the peer
// otherStub is a Stub of the peer, already intialised
func (stub *Stub) MockPeerChaincode(invokableChaincodeName string, otherStub *Stub) {
	stub.Invokables[invokableChaincodeName] = otherStub
}

// MockPeerChaincodeWithChannel registers a peer chaincode with this Stub
func (stub *Stub) MockPeerChaincodeWithChannel(invokableChaincodeName string, otherStub *Stub, channel string) {
	// Internally we use chaincode name as a composite name
	if channel != "" {
		invokableChaincodeName = invokableChaincodeName + "/" + channel
	}

	stub.Invokables[invokableChaincodeName] = otherStub
}

// MockInit initializes this chaincode,  also starts and ends a transaction.
func (stub *Stub) MockInit(uuid string, args [][]byte) pb.Response {
	stub.Args = args
	stub.MockTransactionStart(uuid)
	if stub.cc == nil {
		panic(errors.New("can't init stub (shim.Chaincode) when stub.cc is nil"))
	}
	res := stub.cc.Init(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

// MockInvoke invokes this chaincode, also starts and ends a transaction.
func (stub *Stub) MockInvoke(uuid string, args [][]byte) pb.Response {
	stub.Args = args
	stub.MockTransactionStart(uuid)
	if stub.cc == nil {
		panic(errors.New("can't invoke stub (shim.Chaincode) when stub.cc is nil"))
	}
	res := stub.cc.Invoke(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

// GetDecorations returns the transaction decorations.
func (stub *Stub) GetDecorations() map[string][]byte {
	return stub.Decorations
}

// MockInvokeWithSignedProposal invokes this chaincode, also starts and ends a transaction.
func (stub *Stub) MockInvokeWithSignedProposal(uuid string, args [][]byte, sp *pb.SignedProposal) pb.Response {
	var (
		proposal pb.Proposal
		payload  pb.ChaincodeProposalPayload
	)
	proposalBytes := sp.GetProposalBytes()
	if err := proto.Unmarshal(proposalBytes, &proposal); err != nil {
		return pb.Response{Message: "bad proposal"}
	}
	payloadBytes := proposal.GetPayload()
	if err := proto.Unmarshal(payloadBytes, &payload); err != nil {
		return pb.Response{Message: "bad payload"}
	}
	stub.transientMap = payload.GetTransientMap()
	stub.Args = args
	stub.MockTransactionStart(uuid)
	stub.signedProposal = sp
	if stub.cc == nil {
		panic(errors.New("can't invoke stub (shim.Chaincode) when stub.cc is nil"))
	}
	res := stub.cc.Invoke(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

// GetPrivateData returns the value of the specified `key` from the specified `collection`.
func (stub *Stub) GetPrivateData(collection string, key string) ([]byte, error) {
	m, in := stub.PvtState[collection]

	if !in {
		return nil, nil
	}

	return m[key], nil
}

// GetPrivateDataHash returns the hash of the specified `key` from the specified `collection`.
func (stub *Stub) GetPrivateDataHash(_, _ string) ([]byte, error) { // collection, key
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetPrivateDataHash")
}

// PutPrivateData puts the specified `key` and `value` into the transaction's
func (stub *Stub) PutPrivateData(collection string, key string, value []byte) error {
	m, in := stub.PvtState[collection]
	if !in {
		stub.PvtState[collection] = make(map[string][]byte)
		m = stub.PvtState[collection]
	}

	m[key] = value

	return nil
}

// DelPrivateData removes the specified `key` and its value from the specified `collection`
func (stub *Stub) DelPrivateData(_, _ string) error { // collection, key
	return fmt.Errorf(ErrFuncNotImplemented, "DelPrivateData")
}

// GetPrivateDataByRange returns a range iterator over a set of keys in the
func (stub *Stub) GetPrivateDataByRange(_, _, _ string) (shim.StateQueryIteratorInterface, error) { // collection, startKey, endKey
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetPrivateDataByRange")
}

// GetPrivateDataByPartialCompositeKey returns an iterator over a set of keys
func (stub *Stub) GetPrivateDataByPartialCompositeKey(_, _ string, _ []string) (shim.StateQueryIteratorInterface, error) { // collection, objectType, attributes
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetPrivateDataByPartialCompositeKey")
}

// GetPrivateDataQueryResult performs a "rich" query against a given private
func (stub *Stub) GetPrivateDataQueryResult(_, _ string) (shim.StateQueryIteratorInterface, error) { // collection, query
	// Not implemented since the mock engine does not have a query engine.
	// However, a very simple query engine that supports string matching
	// could be implemented to test that the framework supports queries
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetPrivateDataQueryResult")
}

// PurgePrivateData records the specified keys in the private data collection
func (stub *Stub) PurgePrivateData(_, _ string) error {
	return fmt.Errorf(ErrFuncNotImplemented, "PurgePrivateData")
}

// GetState retrieves the value for a given key from the Ledger
func (stub *Stub) GetState(key string) ([]byte, error) {
	value := stub.State[key]
	stub.logger.Debug("Stub", stub.Name, "Getting", key, value)
	return value, nil
}

// PutState writes the specified `value` and `key` into the Ledger.
func (stub *Stub) PutState(key string, value []byte) error {
	return stub.putState(key, value, true)
}

// PutState writes the specified `value` and `key` into the Ledger.
func (stub *Stub) putState(key string, value []byte, checkTxID bool) error {
	if checkTxID && stub.TxID == "" {
		err := errors.New("cannot PutState without a transactions - call stub.MockTransactionStart()?")
		stub.logger.Errorf("%+v", err)
		return err
	}

	// If the value is nil or empty, delete the key
	if len(value) == 0 {
		stub.logger.Debug("Stub", stub.Name, "PutState called, but value is nil or empty. Delete ", key)
		return stub.DelState(key)
	}

	stub.logger.Debug("Stub", stub.Name, "Putting", key, value)
	stub.State[key] = value

	// insert key into ordered list of keys
OuterLoop:
	for elem := stub.Keys.Front(); elem != nil; elem = elem.Next() {
		elemValue, ok := elem.Value.(string)
		if !ok {
			err := errors.New("cannot requireion elem to string")
			stub.logger.Errorf("%+v", err)
			return err
		}
		comp := strings.Compare(key, elemValue)
		stub.logger.Debug("Stub", stub.Name, "Compared", key, elemValue, " and got ", comp)
		switch {
		case comp < 0:
			stub.Keys.InsertBefore(key, elem)
			stub.logger.Debug("Stub", stub.Name, "Key", key, " inserted before", elem.Value)
			break OuterLoop
		case comp == 0:
			stub.logger.Debug("Stub", stub.Name, "Key", key, "already in State")
			break OuterLoop
		default:
			if elem.Next() == nil {
				stub.Keys.PushBack(key)
				stub.logger.Debug("Stub", stub.Name, "Key", key, "appended")
				break OuterLoop
			}
		}
	}

	// special case for empty Keys list
	if stub.Keys.Len() == 0 {
		stub.Keys.PushFront(key)
		stub.logger.Debug("Stub", stub.Name, "Key", key, "is first element in list")
	}

	return nil
}

// PutBalanceToState writes the specified `value` and `key` into the Ledger.
func (stub *Stub) PutBalanceToState(key string, balance *big.Int) error {
	value := balance.Bytes()
	return stub.putState(key, value, false)
}

// DelState removes the specified `key` and its value from the Ledger.
func (stub *Stub) DelState(key string) error {
	stub.logger.Debug("Stub", stub.Name, "Deleting", key, stub.State[key])
	delete(stub.State, key)

	for elem := stub.Keys.Front(); elem != nil; elem = elem.Next() {
		el, ok := elem.Value.(string)
		if !ok {
			return errors.New("type requireion failed")
		}
		if strings.Compare(key, el) == 0 {
			stub.Keys.Remove(elem)
		}
	}

	return nil
}

// GetStateByRange returns a range iterator over a set of keys in the Ledger.
func (stub *Stub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	if err := validateSimpleKeys(startKey, endKey); err != nil {
		return nil, err
	}
	return NewMockStateRangeQueryIterator(stub, startKey, endKey), nil
}

// GetQueryResult function can be invoked by a chaincode to perform a
// rich query against state database.  Only supported by state database implementations
// that support rich query.  The query string is in the syntax of the underlying
// state database. An iterator is returned which can be used to iterate (next) over
// the query result set
func (stub *Stub) GetQueryResult(_ string) (shim.StateQueryIteratorInterface, error) { // query
	// Not implemented since the mock engine does not have a query engine.
	// However, a very simple query engine that supports string matching
	// could be implemented to test that the framework supports queries
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetQueryResult")
}

// GetHistoryForKey function can be invoked by a chaincode to return a history of
// key values across time. GetHistoryForKey is intended to be used for read-only queries.
func (stub *Stub) GetHistoryForKey(_ string) (shim.HistoryQueryIteratorInterface, error) { // key
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetHistoryForKey")
}

// GetStateByPartialCompositeKey function can be invoked by a chaincode to query the
// state based on a given partial composite key. This function returns an
// iterator which can be used to iterate over all composite keys whose prefix
// matches the given partial composite key. This function should be used only for
// a partial composite key. For a full composite key, an iter with empty response
// would be returned.
func (stub *Stub) GetStateByPartialCompositeKey(objectType string, attributes []string) (shim.StateQueryIteratorInterface, error) {
	partialCompositeKey, err := stub.CreateCompositeKey(objectType, attributes)
	if err != nil {
		return nil, err
	}
	return NewMockStateRangeQueryIterator(stub, partialCompositeKey, partialCompositeKey+string(utf8.MaxRune)), nil
}

// CreateCompositeKey combines the list of attributes
// to form a composite key.
func (stub *Stub) CreateCompositeKey(objectType string, attributes []string) (string, error) {
	return createCompositeKey(objectType, attributes)
}

// SplitCompositeKey splits the composite key into attributes
// on which the composite key was formed.
func (stub *Stub) SplitCompositeKey(compositeKey string) (string, []string, error) {
	return splitCompositeKey(compositeKey)
}

// GetStateByRangeWithPagination returns a range iterator over a set of keys in the Ledger.
func (stub *Stub) GetStateByRangeWithPagination(
	startKey, endKey string,
	pageSize int32,
	bookmark string,
) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	if bookmark != "" {
		startKey = bookmark
	}

	if err := validateSimpleKeys(startKey, endKey); err != nil {
		return nil, nil, err
	}

	iter := NewMockStateRangeQueryIterator(stub, startKey, endKey)
	defer func() {
		_ = iter.Close()
	}()

	count := int32(0)
	res := make([]string, 0, pageSize)
	for iter.HasNext() && count < pageSize {
		val, err := iter.Next()
		if err != nil {
			return nil, nil, err
		}
		res = append(res, val.GetKey())
		count++
	}

	b := ""
	if iter.HasNext() {
		v, err := iter.Next()
		if err != nil {
			return nil, nil, err
		}

		b = v.GetKey()
	}

	m := &pb.QueryResponseMetadata{
		FetchedRecordsCount: count,
		Bookmark:            b,
	}

	return NewMockStateRangeQueryWithPaginationIterator(stub, res), m, nil
}

// GetStateByPartialCompositeKeyWithPagination returns a range iterator over a set of keys in the Ledger.
func (stub *Stub) GetStateByPartialCompositeKeyWithPagination(
	objectType string,
	keys []string,
	pageSize int32,
	bookmark string,
) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	partialCompositeKey, err := stub.CreateCompositeKey(objectType, keys)
	if err != nil {
		return nil, nil, err
	}

	if bookmark == "" {
		bookmark = partialCompositeKey
	}

	iter := NewMockStateRangeQueryIterator(stub, bookmark, partialCompositeKey+string(utf8.MaxRune))
	defer func() {
		_ = iter.Close()
	}()

	count := int32(0)
	res := make([]string, 0, pageSize)
	for iter.HasNext() && count < pageSize {
		val, err := iter.Next()
		if err != nil {
			return nil, nil, err
		}
		res = append(res, val.GetKey())
		count++
	}

	b := ""
	if iter.HasNext() {
		v, err := iter.Next()
		if err != nil {
			return nil, nil, err
		}

		b = v.GetKey()
	}

	m := &pb.QueryResponseMetadata{
		FetchedRecordsCount: count,
		Bookmark:            b,
	}

	return NewMockStateRangeQueryWithPaginationIterator(stub, res), m, nil
}

// GetQueryResultWithPagination performs a "rich" query against a given state database.
func (stub *Stub) GetQueryResultWithPagination(
	_ string, // query
	_ int32, // pageSize
	_ string, // bookmark
) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	return nil, nil, fmt.Errorf(ErrFuncNotImplemented, "GetQueryResultWithPagination")
}

// InvokeChaincode calls a peered chaincode.
// E.g. stub1.InvokeChaincode("stub2Hash", funcArgs, channel)
// Before calling this make sure to create another Stub stub2, call stub2.MockInit(uuid, func, Args)
// and register it with stub1 by calling stub1.MockPeerChaincode("stub2Hash", stub2)
func (stub *Stub) InvokeChaincode(chaincodeName string, args [][]byte, channel string) pb.Response {
	// Internally we use chaincode name as a composite name
	if channel != "" {
		chaincodeName = chaincodeName + "/" + channel
	}

	otherStub := stub.Invokables[chaincodeName]
	stub.logger.Debug("Stub", stub.Name, "Invoking peer chaincode", otherStub.Name, args)
	//	function, strings := getFuncArgs(Args)
	res := otherStub.MockInvoke(stub.TxID, args)
	stub.logger.Debug("Stub", stub.Name, "Invoked peer chaincode", otherStub.Name, "got", fmt.Sprintf("%+v", res))
	return res
}

// SetCreator sets creator
func (stub *Stub) SetCreator(creator []byte) {
	stub.creator = creator
}

// SetCreatorCert sets creator cert
func (stub *Stub) SetCreatorCert(creatorMSP string, creatorCert []byte) error {
	creator, err := BuildCreator(creatorMSP, creatorCert)
	if err != nil {
		return err
	}
	stub.creator = creator
	return nil
}

func BuildCreator(creatorMSP string, creatorCert []byte) ([]byte, error) {
	pemblock := &pem.Block{Type: "CERTIFICATE", Bytes: creatorCert}
	pemBytes := pem.EncodeToMemory(pemblock)
	if pemBytes == nil {
		return nil, errors.New("encoding of identity failed")
	}

	creator := &msp.SerializedIdentity{Mspid: creatorMSP, IdBytes: pemBytes}
	marshaledIdentity, err := proto.Marshal(creator)
	if err != nil {
		return nil, err
	}
	return marshaledIdentity, nil
}

// GetCreator returns creator.
func (stub *Stub) GetCreator() ([]byte, error) {
	return stub.creator, nil
}

// GetTransient returns transient. Not implemented
func (stub *Stub) GetTransient() (map[string][]byte, error) {
	return stub.transientMap, nil
}

// GetBinding returns binding. Not implemented
func (stub *Stub) GetBinding() ([]byte, error) {
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetBinding")
}

// GetSignedProposal returns proposal. Not implemented
func (stub *Stub) GetSignedProposal() (*pb.SignedProposal, error) {
	return stub.signedProposal, nil
}

func (stub *Stub) setSignedProposal(sp *pb.SignedProposal) {
	stub.signedProposal = sp
}

// GetArgsSlice returns Args slice. Not implemented
func (stub *Stub) GetArgsSlice() ([]byte, error) {
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetArgsSlice")
}

func (stub *Stub) setTxTimestamp(time *timestamp.Timestamp) {
	stub.TxTimestamp = time
}

// GetTxTimestamp returns timestamp.
func (stub *Stub) GetTxTimestamp() (*timestamp.Timestamp, error) {
	if stub.TxTimestamp == nil {
		return nil, errors.New("timestamp was not set")
	}
	return stub.TxTimestamp, nil
}

// SetEvent allows the chaincode to set an event
func (stub *Stub) SetEvent(name string, payload []byte) error {
	stub.ChaincodeEventsChannel <- &pb.ChaincodeEvent{EventName: name, Payload: payload}
	return nil
}

// SetStateValidationParameter sets the state validation parameter for the given key
func (stub *Stub) SetStateValidationParameter(key string, ep []byte) error {
	return stub.SetPrivateDataValidationParameter("", key, ep)
}

// GetStateValidationParameter gets the state validation parameter for the given key
func (stub *Stub) GetStateValidationParameter(key string) ([]byte, error) {
	return stub.GetPrivateDataValidationParameter("", key)
}

// SetPrivateDataValidationParameter sets the private data validation parameter for the given collection and key
func (stub *Stub) SetPrivateDataValidationParameter(collection, key string, ep []byte) error {
	m, in := stub.EndorsementPolicies[collection]
	if !in {
		stub.EndorsementPolicies[collection] = make(map[string][]byte)
		m = stub.EndorsementPolicies[collection]
	}

	m[key] = ep
	return nil
}

// GetPrivateDataValidationParameter gets the private data validation parameter for the given collection and key
func (stub *Stub) GetPrivateDataValidationParameter(collection, key string) ([]byte, error) {
	m, in := stub.EndorsementPolicies[collection]

	if !in {
		return nil, nil
	}

	return m[key], nil
}

/*****************************
 Range Query Iterator
*****************************/

// StateRangeQueryIterator is an interface that is used to iterate over a set of keys
type StateRangeQueryIterator struct {
	Closed   bool
	Stub     *Stub
	StartKey string
	EndKey   string
	Current  *list.Element
}

// HasNext returns true if the range query iterator contains additional keys
// and values.
func (iter *StateRangeQueryIterator) HasNext() bool {
	if iter.Closed {
		// previously called Close()
		iter.Stub.logger.Debug("HasNext() but already closed")
		return false
	}

	if iter.Current == nil {
		iter.Stub.logger.Error("HasNext() couldn't get Current")
		return false
	}

	current := iter.Current
	for current != nil {
		// if this is an open-ended query for all keys, return true
		if iter.StartKey == "" && iter.EndKey == "" {
			return true
		}
		curStr, _ := current.Value.(string)
		comp1 := strings.Compare(curStr, iter.StartKey)
		comp2 := strings.Compare(curStr, iter.EndKey)
		if comp1 >= 0 {
			if comp2 < 0 {
				iter.Stub.logger.Debug("HasNext() got next")
				return true
			}

			iter.Stub.logger.Debug("HasNext() but no next")
			return false
		}
		current = current.Next()
	}

	// we've reached the end of the underlying values
	iter.Stub.logger.Debug("HasNext() but no next")
	return false
}

// Next returns the next key and value in the range query iterator.
func (iter *StateRangeQueryIterator) Next() (*queryresult.KV, error) {
	if iter.Closed {
		err := errors.New("StateRangeQueryIterator.Next() called after Close()")
		iter.Stub.logger.Errorf("%+v", err)
		return nil, err
	}

	if !iter.HasNext() {
		err := errors.New("StateRangeQueryIterator.Next() called when it does not HaveNext()")
		iter.Stub.logger.Errorf("%+v", err)
		return nil, err
	}

	for iter.Current != nil {
		curStr, _ := iter.Current.Value.(string)
		comp1 := strings.Compare(curStr, iter.StartKey)
		comp2 := strings.Compare(curStr, iter.EndKey)
		// compare to start and end keys. or, if this is an open-ended query for
		// all keys, it should always return the key and value
		if (comp1 >= 0 && comp2 < 0) || (iter.StartKey == "" && iter.EndKey == "") {
			key, _ := iter.Current.Value.(string)

			value, err := iter.Stub.GetState(key)
			iter.Current = iter.Current.Next()
			return &queryresult.KV{Key: key, Value: value}, err
		}
		iter.Current = iter.Current.Next()
	}
	err := errors.New("StateRangeQueryIterator.Next() went past end of range")
	iter.Stub.logger.Errorf("%+v", err)
	return nil, err
}

// Close closes the range query iterator. This should be called when done
// reading from the iterator to free up resources.
func (iter *StateRangeQueryIterator) Close() error {
	if iter.Closed {
		err := errors.New("StateRangeQueryIterator.Close() called after Close()")
		iter.Stub.logger.Errorf("%+v", err)
		return err
	}

	iter.Closed = true
	return nil
}

// Print prints the StateRangeQueryIterator
func (iter *StateRangeQueryIterator) Print() {
	iter.Stub.logger.Debug("StateRangeQueryIterator {")
	iter.Stub.logger.Debug("Closed?", iter.Closed)
	iter.Stub.logger.Debug("Stub", iter.Stub)
	iter.Stub.logger.Debug("StartKey", iter.StartKey)
	iter.Stub.logger.Debug("EndKey", iter.EndKey)
	iter.Stub.logger.Debug("Current", iter.Current)
	iter.Stub.logger.Debug("HasNext?", iter.HasNext())
	iter.Stub.logger.Debug("}")
}

// NewMockStateRangeQueryIterator - Constructor for a StateRangeQueryIterator
func NewMockStateRangeQueryIterator(stub *Stub, startKey string, endKey string) *StateRangeQueryIterator {
	stub.logger.Debug("NewMockStateRangeQueryIterator(", stub, startKey, endKey, ")")
	iter := new(StateRangeQueryIterator)
	iter.Closed = false
	iter.Stub = stub
	iter.StartKey = startKey
	iter.EndKey = endKey
	iter.Current = stub.Keys.Front()

	iter.Print()

	return iter
}

/*****************************
 Range Query Iterator With Pagination
*****************************/

// StateRangeQueryWithPaginationIterator is an interface that is used to iterate over a set of keys
type StateRangeQueryWithPaginationIterator struct {
	Closed   bool
	Stub     *Stub
	Elements []string
}

// HasNext returns true if the range query iterator contains additional keys
// and values.
func (iter *StateRangeQueryWithPaginationIterator) HasNext() bool {
	if iter.Closed {
		// previously called Close()
		iter.Stub.logger.Debug("HasNext() but already closed")
		return false
	}

	if len(iter.Elements) == 0 {
		iter.Stub.logger.Debug("HasNext() but no next")
		return false
	}

	iter.Stub.logger.Debug("HasNext() got next")
	return true
}

// Next returns the next key and value in the range query iterator.
func (iter *StateRangeQueryWithPaginationIterator) Next() (*queryresult.KV, error) {
	if iter.Closed {
		err := errors.New("StateRangeQueryWithPaginationIterator.Next() called after Close()")
		iter.Stub.logger.Errorf("%+v", err)
		return nil, err
	}

	if !iter.HasNext() {
		err := errors.New("StateRangeQueryWithPaginationIterator.Next() called when it does not HaveNext()")
		iter.Stub.logger.Errorf("%+v", err)
		return nil, err
	}

	key := iter.Elements[0]
	value, err := iter.Stub.GetState(key)

	iter.Elements = iter.Elements[1:]
	return &queryresult.KV{Key: key, Value: value}, err
}

// Close closes the range query iterator. This should be called when done
// reading from the iterator to free up resources.
func (iter *StateRangeQueryWithPaginationIterator) Close() error {
	if iter.Closed {
		err := errors.New("StateRangeQueryWithPaginationIterator.Close() called after Close()")
		iter.Stub.logger.Errorf("%+v", err)
		return err
	}

	iter.Elements = nil
	iter.Closed = true
	return nil
}

// NewMockStateRangeQueryWithPaginationIterator - Constructor for a StateRangeQueryWithPaginationIterator
func NewMockStateRangeQueryWithPaginationIterator(stub *Stub, elements []string) *StateRangeQueryWithPaginationIterator {
	stub.logger.Debug("NewMockStateRangeQueryWithPaginationIterator(", stub, ")")

	iter := &StateRangeQueryWithPaginationIterator{
		Closed:   false,
		Stub:     stub,
		Elements: elements,
	}

	return iter
}

const (
	minUnicodeRuneValue   = 0            // U+0000
	maxUnicodeRuneValue   = utf8.MaxRune // U+10FFFF - maximum (and unallocated) code point
	compositeKeyNamespace = "\x00"
	// emptyKeySubstitute    = "\x01"
)

func validateSimpleKeys(simpleKeys ...string) error {
	for _, key := range simpleKeys {
		if len(key) > 0 && key[0] == compositeKeyNamespace[0] {
			return errors.New("first character of the key [" + key + "] contains a null character which is not allowed")
		}
	}
	return nil
}

func createCompositeKey(objectType string, attributes []string) (string, error) {
	if err := validateCompositeKeyAttribute(objectType); err != nil {
		return "", err
	}
	ck := compositeKeyNamespace + objectType + string(rune(minUnicodeRuneValue))
	for _, att := range attributes {
		if err := validateCompositeKeyAttribute(att); err != nil {
			return "", err
		}
		ck += att + string(rune(minUnicodeRuneValue))
	}
	return ck, nil
}

func splitCompositeKey(compositeKey string) (string, []string, error) {
	componentIndex := 1
	var components []string
	for i := 1; i < len(compositeKey); i++ {
		if compositeKey[i] == minUnicodeRuneValue {
			components = append(components, compositeKey[componentIndex:i])
			componentIndex = i + 1
		}
	}
	return components[0], components[1:], nil
}

func validateCompositeKeyAttribute(str string) error {
	if !utf8.ValidString(str) {
		return errors.New("not a valid utf8 string: [" + str + "]")
	}
	for index, runeValue := range str {
		if runeValue == minUnicodeRuneValue || runeValue == maxUnicodeRuneValue {
			return fmt.Errorf(`input contain unicode %#U starting at position [%d]. %#U and %#U are not allowed in the input attribute of a composite key`,
				runeValue, index, minUnicodeRuneValue, maxUnicodeRuneValue)
		}
	}
	return nil
}

// CreateUtcTimestamp returns a Google/protobuf/Timestamp in UTC
func createUtcTimestamp() *timestamp.Timestamp {
	now := time.Now().UTC()
	secs := now.Unix()
	nanos := int32(now.UnixNano() - (secs * 1000000000)) //nolint:gomnd
	return &(timestamp.Timestamp{Seconds: secs, Nanos: nanos})
}

// SetAdminCreatorCert sets admin certificate as creator certificate.
func (stub *Stub) SetAdminCreatorCert(msp string) error {
	// assume adminCert has valid base64 encoded certificate
	cert, _ := base64.StdEncoding.DecodeString(adminCert)
	if err := stub.SetCreatorCert(msp, cert); err != nil {
		return fmt.Errorf("setting creator: %w", err)
	}

	return nil
}

// SetDefaultCreatorCert sets default (not admin) certificate as creator certificate.
func (stub *Stub) SetDefaultCreatorCert(msp string) error {
	// assume adminCert has valid base64 encoded certificate
	cert, _ := base64.StdEncoding.DecodeString(defaultCert)
	if err := stub.SetCreatorCert(msp, cert); err != nil {
		return fmt.Errorf("setting creator: %w", err)
	}

	return nil
}

func (stub *Stub) AddAccountingRecord(
	token string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) {
	stub.logger.Infof(
		"AddAccountingRecord: token: %s, from: %v, to: %v, amount: %s, reason: %s",
		token,
		from,
		to,
		amount.String(),
		reason,
	)
}
