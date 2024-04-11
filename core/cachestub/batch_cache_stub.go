package cachestub

import (
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type BatchWriteCache struct {
	shim.ChaincodeStubInterface
	batchCache map[string]*proto.WriteElement
	Swaps      []*proto.Swap
	MultiSwaps []*proto.MultiSwap
}

func NewBatchCacheStub(stub shim.ChaincodeStubInterface) *BatchWriteCache {
	return &BatchWriteCache{
		ChaincodeStubInterface: stub,
		batchCache:             make(map[string]*proto.WriteElement),
	}
}

// GetState returns state from BatchWriteCache cache or, if absent, from chaincode state
func (bs *BatchWriteCache) GetState(key string) ([]byte, error) {
	existsElement, ok := bs.batchCache[key]
	if ok {
		return existsElement.Value, nil
	}
	return bs.ChaincodeStubInterface.GetState(key)
}

// PutState puts state to a BatchWriteCache cache
func (bs *BatchWriteCache) PutState(key string, value []byte) error {
	bs.batchCache[key] = &proto.WriteElement{Key: key, Value: value}
	return nil
}

// Commit puts state from a BatchWriteCache cache to the chaincode state
func (bs *BatchWriteCache) Commit() error {
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

// DelState - marks state in BatchWriteCache cache as deleted
func (bs *BatchWriteCache) DelState(key string) error {
	bs.batchCache[key] = &proto.WriteElement{Key: key, IsDeleted: true}
	return nil
}
