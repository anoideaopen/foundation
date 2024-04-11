package cachestub

import (
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type BatchCacheStub struct {
	shim.ChaincodeStubInterface
	batchWriteCache map[string]*proto.WriteElement
	Swaps           []*proto.Swap
	MultiSwaps      []*proto.MultiSwap
}

func NewBatchCacheStub(stub shim.ChaincodeStubInterface) *BatchCacheStub {
	return &BatchCacheStub{
		ChaincodeStubInterface: stub,
		batchWriteCache:        make(map[string]*proto.WriteElement),
	}
}

// GetState returns state from BatchCacheStub cache or, if absent, from chaincode state
func (bs *BatchCacheStub) GetState(key string) ([]byte, error) {
	existsElement, ok := bs.batchWriteCache[key]
	if ok {
		return existsElement.Value, nil
	}
	return bs.ChaincodeStubInterface.GetState(key)
}

// PutState puts state to a BatchCacheStub cache
func (bs *BatchCacheStub) PutState(key string, value []byte) error {
	bs.batchWriteCache[key] = &proto.WriteElement{Key: key, Value: value}
	return nil
}

// Commit puts state from a BatchCacheStub cache to the chaincode state
func (bs *BatchCacheStub) Commit() error {
	for key, element := range bs.batchWriteCache {
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

// DelState - marks state in BatchCacheStub cache as deleted
func (bs *BatchCacheStub) DelState(key string) error {
	bs.batchWriteCache[key] = &proto.WriteElement{Key: key, IsDeleted: true}
	return nil
}
