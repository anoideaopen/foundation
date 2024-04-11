package cachestub

import (
	"sort"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
)

type TxWriteCache struct {
	*BatchWriteCache
	txID       string
	txCache    map[string]*proto.WriteElement
	events     map[string][]byte
	Accounting []*proto.AccountingRecord
}

func (bs *BatchWriteCache) NewTxCacheStub(txID string) *TxWriteCache {
	return &TxWriteCache{
		BatchWriteCache: bs,
		txID:            txID,
		txCache:         make(map[string]*proto.WriteElement),
		events:          make(map[string][]byte),
	}
}

// GetTxID returns TxWriteCache transaction ID
func (bts *TxWriteCache) GetTxID() string {
	return bts.txID
}

// GetState returns state from TxWriteCache cache or, if absent, from batchState cache
func (bts *TxWriteCache) GetState(key string) ([]byte, error) {
	existsElement, ok := bts.txCache[key]
	if ok {
		return existsElement.Value, nil
	}
	return bts.BatchWriteCache.GetState(key)
}

// PutState puts state to the TxWriteCache's cache
func (bts *TxWriteCache) PutState(key string, value []byte) error {
	bts.txCache[key] = &proto.WriteElement{Value: value}
	return nil
}

// SetEvent sets payload to a TxWriteCache events
func (bts *TxWriteCache) SetEvent(name string, payload []byte) error {
	bts.events[name] = payload
	return nil
}

func (bts *TxWriteCache) AddAccountingRecord(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) {
	bts.Accounting = append(bts.Accounting, &proto.AccountingRecord{
		Token:     token,
		Sender:    from.Bytes(),
		Recipient: to.Bytes(),
		Amount:    amount.Bytes(),
		Reason:    reason,
	})
}

// Commit puts state from a TxWriteCache cache to the BatchWriteCache cache
func (bts *TxWriteCache) Commit() ([]*proto.WriteElement, []*proto.Event) {
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

// DelState marks state in TxWriteCache as deleted
func (bts *TxWriteCache) DelState(key string) error {
	bts.txCache[key] = &proto.WriteElement{Key: key, IsDeleted: true}
	return nil
}
