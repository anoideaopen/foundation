package cachestub

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	pb "google.golang.org/protobuf/proto"
)

type BatchCacheStub struct {
	shim.ChaincodeStubInterface
	batchWriteCache   map[string]*proto.WriteElement
	batchReadeCache   map[string]*proto.WriteElement
	invokeResultCache map[string]*peer.Response
	Swaps             []*proto.Swap
	MultiSwaps        []*proto.MultiSwap
}

func NewBatchCacheStub(stub shim.ChaincodeStubInterface) *BatchCacheStub {
	return &BatchCacheStub{
		ChaincodeStubInterface: stub,
		batchWriteCache:        make(map[string]*proto.WriteElement),
		batchReadeCache:        make(map[string]*proto.WriteElement),
		invokeResultCache:      make(map[string]*peer.Response),
	}
}

// GetState returns state from BatchCacheStub cache or, if absent, from chaincode state
func (bs *BatchCacheStub) GetState(key string) ([]byte, error) {
	if existsElement, ok := bs.batchWriteCache[key]; ok {
		return existsElement.GetValue(), nil
	}

	if existsElement, ok := bs.batchReadeCache[key]; ok {
		return existsElement.GetValue(), nil
	}

	value, err := bs.ChaincodeStubInterface.GetState(key)
	if err != nil {
		return nil, err
	}

	bs.batchReadeCache[key] = &proto.WriteElement{Key: key, Value: value}

	return value, nil
}

// PutState puts state to a BatchCacheStub cache
func (bs *BatchCacheStub) PutState(key string, value []byte) error {
	bs.batchWriteCache[key] = &proto.WriteElement{Key: key, Value: value}
	return nil
}

// Commit puts state from a BatchCacheStub cache to the chaincode state
func (bs *BatchCacheStub) Commit() error {
	for key, element := range bs.batchWriteCache {
		if element.GetIsDeleted() {
			if err := bs.ChaincodeStubInterface.DelState(key); err != nil {
				return err
			}
		} else {
			if err := bs.ChaincodeStubInterface.PutState(key, element.GetValue()); err != nil {
				return err
			}
		}
	}
	return nil
}

// DelState - marks state in BatchCacheStub cache as deleted
func (bs *BatchCacheStub) DelState(key string) error {
	bs.batchWriteCache[key] = &proto.WriteElement{Key: key, IsDeleted: true}
	return nil
}

func (bs *BatchCacheStub) InvokeChaincode(chaincodeName string, args [][]byte, channel string) *peer.Response {
	var key string
	if string(args[0]) != "getAccountsInfo" {
		key = bs.makeKeyByte(channel, chaincodeName, args)

		if resp, ok := bs.invokeResultCache[key]; ok {
			return resp
		}
	}

	resp := bs.ChaincodeStubInterface.InvokeChaincode(chaincodeName, args, channel)

	if resp.GetStatus() == http.StatusOK && len(resp.GetPayload()) != 0 {
		switch string(args[0]) {
		case "checkKeys":
			bs.insertCacheCheckKeys(channel, chaincodeName, resp)
			bs.invokeResultCache[key] = resp
		case "getAccountsInfo":
			bs.insertCacheGetAccountsInfo(channel, chaincodeName, args[1:], resp)
		default:
			bs.invokeResultCache[key] = resp
		}
	}

	return resp
}

func (bs *BatchCacheStub) insertCacheCheckKeys(
	channel string,
	chaincodeName string,
	resp *peer.Response,
) {
	addrMsg := &proto.AclResponse{}
	if err := pb.Unmarshal(resp.GetPayload(), addrMsg); err != nil {
		return
	}

	addr := addrMsg.GetAddress().GetAddress().AddrString()

	address, err := pb.Marshal(addrMsg.GetAddress().GetAddress())
	if err != nil {
		return
	}
	keyCheck := channel + chaincodeName + "checkAddress" + addr
	bs.invokeResultCache[keyCheck] = &peer.Response{
		Status:  http.StatusOK,
		Payload: address,
	}

	accinfo, err := json.Marshal(addrMsg.GetAccount())
	if err != nil {
		return
	}
	keyAccInfo := channel + chaincodeName + "getAccountInfo" + addr
	bs.invokeResultCache[keyAccInfo] = &peer.Response{
		Status:  http.StatusOK,
		Payload: accinfo,
	}
}

func (bs *BatchCacheStub) insertCacheGetAccountsInfo(
	channel string,
	chaincodeName string,
	args [][]byte,
	resp *peer.Response,
) {
	var responses []*peer.Response
	err := json.Unmarshal(resp.GetPayload(), &responses)
	if err != nil {
		return
	}
	if len(responses) != len(args) {
		return
	}

	skip := make([]int, 0, len(responses))

	for i, arg := range args {
		if responses[i].GetStatus() != http.StatusOK || len(responses[i].GetPayload()) == 0 {
			continue
		}

		var argsTmp []string
		err = json.Unmarshal(arg, &argsTmp)
		if err != nil {
			continue
		}

		if argsTmp[0] != "checkKeys" {
			skip = append(skip, i)
			continue
		}

		key := bs.makeKeyString(channel, chaincodeName, argsTmp)
		if _, ok := bs.invokeResultCache[key]; ok {
			continue
		}

		bs.insertCacheCheckKeys(channel, chaincodeName, responses[i])
		bs.invokeResultCache[key] = responses[i]
	}

	for _, i := range skip {
		var argsTmp []string
		err = json.Unmarshal(args[i], &argsTmp)
		if err != nil {
			continue
		}

		key := bs.makeKeyString(channel, chaincodeName, argsTmp)
		if _, ok := bs.invokeResultCache[key]; ok {
			continue
		}

		bs.invokeResultCache[key] = responses[i]
	}
}

func (bs *BatchCacheStub) makeKeyByte(
	channel string,
	chaincodeName string,
	args [][]byte,
) string {
	keys := []string{channel, chaincodeName}
	for _, arg := range args {
		keys = append(keys, string(arg))
	}
	return strings.Join(keys, "")
}

func (bs *BatchCacheStub) makeKeyString(
	channel string,
	chaincodeName string,
	args []string,
) string {
	keys := []string{channel, chaincodeName}
	keys = append(keys, args...)
	return strings.Join(keys, "")
}
