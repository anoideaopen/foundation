package cachestub

import (
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

const retry = 5

type BatchCacheStub struct {
	shim.ChaincodeStubInterface
	batchWriteCache   map[string]*proto.WriteElement
	batchReadeCache   map[string]*proto.WriteElement
	readLock          sync.RWMutex
	invokeResultCache map[string]pb.Response
	invokeLock        sync.RWMutex
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

	bs.readLock.RLock()
	existsElement, ok := bs.batchReadeCache[key]
	bs.readLock.RUnlock()
	if ok {
		return existsElement.GetValue(), nil
	}

	value, err := bs.ChaincodeStubInterface.GetState(key)
	if err != nil {
		return nil, err
	}

	bs.readLock.Lock()
	bs.batchReadeCache[key] = &proto.WriteElement{Key: key, Value: value}
	bs.readLock.Unlock()

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

	bs.invokeLock.RLock()
	result, ok := bs.invokeResultCache[key]
	bs.invokeLock.RUnlock()
	if ok {
		return result
	}

	var resp pb.Response

	bs.invokeLock.Lock()
	for i := 0; i < retry; i++ {
		resp = bs.ChaincodeStubInterface.InvokeChaincode(chaincodeName, args, channel)

		if resp.GetStatus() == http.StatusOK {
			break
		}

		tt := time.Duration(float64(50*time.Millisecond) + 0.2*(rand.Float64()*2-1))
		time.Sleep(tt)
	}

	if resp.GetStatus() == http.StatusOK && len(resp.GetPayload()) != 0 {
		bs.invokeResultCache[key] = resp
	}
	bs.invokeLock.Unlock()

	return resp
}
