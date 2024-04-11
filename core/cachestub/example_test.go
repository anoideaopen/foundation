package cachestub_test

import (
	"fmt"

	"github.com/anoideaopen/foundation/core/cachestub"
)

// Example demonstrates how to use BatchCacheStub and TxCacheStub to perform transactions on a cache.
func Example() {
	// Initialize a mock chaincode stub
	mockStub := newMockStub()

	// Initialize a batch cache stub
	batchCache := cachestub.NewBatchCacheStub(mockStub)

	// Start a transaction to update value of KEY1
	tx1 := batchCache.NewTxCacheStub("tx1")
	newValue := []byte("new_value")
	_ = tx1.PutState("KEY1", newValue)
	tx1.Commit()

	// Start another transaction to update value of KEY2 with the previous value of KEY1
	tx2 := batchCache.NewTxCacheStub("tx2")
	value, _ := tx2.GetState("KEY1")
	_ = tx2.PutState("KEY2", value)
	_ = tx2.DelState("KEY1")
	tx2.Commit()

	// Commit the batch changes
	err := batchCache.Commit()
	if err != nil {
		panic(err)
	}

	// Verify the changes after committing the transaction
	updatedValue1, _ := batchCache.GetState("KEY1")
	updatedValue2, _ := batchCache.GetState("KEY2")

	// Verify the changes after committing transactions and batch
	fmt.Printf(string(updatedValue1))
	// Output:
	fmt.Printf(string(updatedValue2))
	// Output: new_value
}
