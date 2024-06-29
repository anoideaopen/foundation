package cachestub

import (
	"net/http"
	"strings"

	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

type BatchCacheStub struct {
	shim.ChaincodeStubInterface
	batchWriteCache   map[string]*proto.WriteElement
	batchReadeCache   map[string]*proto.WriteElement
	invokeResultCache map[string]pb.Response
	Swaps             []*proto.Swap
	MultiSwaps        []*proto.MultiSwap
}

func NewBatchCacheStub(stub shim.ChaincodeStubInterface) *BatchCacheStub {
	return &BatchCacheStub{
		ChaincodeStubInterface: stub,
		batchWriteCache:        make(map[string]*proto.WriteElement),
		batchReadeCache:        make(map[string]*proto.WriteElement),
		invokeResultCache:      make(map[string]pb.Response),
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

func (bs *BatchCacheStub) InvokeChaincode(chaincodeName string, args [][]byte, channel string) pb.Response {
	keys := []string{channel, chaincodeName}
	for _, arg := range args {
		keys = append(keys, string(arg))
	}
	key := strings.Join(keys, "")

	if result, ok := bs.invokeResultCache[key]; ok {
		return result
	}

	resp := bs.ChaincodeStubInterface.InvokeChaincode(chaincodeName, args, channel)

	if resp.GetStatus() == http.StatusOK && len(resp.GetPayload()) != 0 {
		bs.invokeResultCache[key] = resp
	}

	return resp
}
