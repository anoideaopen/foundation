package core

import (
	"sort"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type batchStub struct {
	shim.ChaincodeStubInterface
	batchCache map[string]*proto.WriteElement
	swaps      []*proto.Swap
	multiSwaps []*proto.MultiSwap
}

func newBatchStub(stub shim.ChaincodeStubInterface) *batchStub {
	return &batchStub{
		ChaincodeStubInterface: stub,
		batchCache:             make(map[string]*proto.WriteElement),
	}
}

// GetState returns state from batchStub cache or, if absent, from chaincode state
func (bs *batchStub) GetState(key string) ([]byte, error) {
	existsElement, ok := bs.batchCache[key]
	if ok {
		return existsElement.Value, nil
	}
	return bs.ChaincodeStubInterface.GetState(key)
}

// PutState puts state to a batchStub cache
func (bs *batchStub) PutState(key string, value []byte) error {
	bs.batchCache[key] = &proto.WriteElement{Key: key, Value: value}
	return nil
}

// Commit puts state from a batchStub cache to the chaincode state
func (bs *batchStub) Commit() error {
	for key, element := range bs.batchCache {
		if element.IsDeleted {
			if err := bs.ChaincodeStubInterface.DelState(key); err != nil {
				return err
			}
		} else {
			if err := bs.ChaincodeStubInterface.PutState(key, element.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

// DelState - marks state in batchStub cache as deleted
func (bs *batchStub) DelState(key string) error {
	bs.batchCache[key] = &proto.WriteElement{Key: key, IsDeleted: true}
	return nil
}

// BatchTxStub is a stub for batch transaction
type BatchTxStub struct {
	*batchStub
	txID       string
	txCache    map[string]*proto.WriteElement
	events     map[string][]byte
	accounting []*proto.AccountingRecord
}

func (bs *batchStub) newTxStub(txID string) *BatchTxStub {
	return &BatchTxStub{
		batchStub: bs,
		txID:      txID,
		txCache:   make(map[string]*proto.WriteElement),
		events:    make(map[string][]byte),
	}
}

// GetTxID returns batchTxStub transaction ID
func (bts *BatchTxStub) GetTxID() string {
	return bts.txID
}

// GetState returns state from batchTxStub cache or, if absent, from batchState cache
func (bts *BatchTxStub) GetState(key string) ([]byte, error) {
	existsElement, ok := bts.txCache[key]
	if ok {
		return existsElement.Value, nil
	}
	return bts.batchStub.GetState(key)
}

// PutState puts state to the batchTxStub's cache
func (bts *BatchTxStub) PutState(key string, value []byte) error {
	bts.txCache[key] = &proto.WriteElement{Value: value}
	return nil
}

// SetEvent sets payload to a batchTxStub events
func (bts *BatchTxStub) SetEvent(name string, payload []byte) error {
	bts.events[name] = payload
	return nil
}

// AddAccountingRecord adds accounting record to a batchTxStub accounting
func (bts *BatchTxStub) AddAccountingRecord(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) {
	bts.accounting = append(bts.accounting, &proto.AccountingRecord{
		Token:     token,
		Sender:    from.Bytes(),
		Recipient: to.Bytes(),
		Amount:    amount.Bytes(),
		Reason:    reason,
	})
}

// Commit puts state from a batchTxStub cache to the batchStub cache
func (bts *BatchTxStub) Commit() ([]*proto.WriteElement, []*proto.Event) {
	writeKeys := make([]string, 0, len(bts.txCache))
	for k, v := range bts.txCache {
		bts.batchCache[k] = v
		writeKeys = append(writeKeys, k)
	}
	sort.Strings(writeKeys)
	writes := make([]*proto.WriteElement, 0, len(writeKeys))
	for _, k := range writeKeys {
		writes = append(writes, &proto.WriteElement{
			Key:       k,
			Value:     bts.txCache[k].Value,
			IsDeleted: bts.txCache[k].IsDeleted,
		})
	}

	eventKeys := make([]string, 0, len(bts.events))
	for k := range bts.events {
		eventKeys = append(eventKeys, k)
	}
	sort.Strings(eventKeys)
	events := make([]*proto.Event, 0, len(eventKeys))
	for _, k := range eventKeys {
		events = append(events, &proto.Event{
			Name:  k,
			Value: bts.events[k],
		})
	}
	return writes, events
}

// DelState marks state in batchTxStub as deleted
func (bts *BatchTxStub) DelState(key string) error {
	bts.txCache[key] = &proto.WriteElement{Key: key, IsDeleted: true}
	return nil
}
