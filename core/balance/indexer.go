package balance

import "github.com/hyperledger/fabric-chaincode-go/shim"

// IMPORTANT: THE INDEXER CAN BE USED AS A TOOL FOR MIGRATING EXISTING
// TOKENS. DETAILS IN README.md.

// IndexCreatedKey is the key used to store the index creation flag.
const IndexCreatedKey = "balance_index_created"

// IndexLastKeyPrefix is the last indexed key
const IndexLastKeyPrefix = "last_indexed_key_"

// Balances per transaction
const limit = 100

// CreateIndex builds an index for states matching the specified balance type.
// It processes records in batches of 100 and continues from where it left off
// in previous executions.
//
// Parameters:
//   - stub: shim.ChaincodeStubInterface - The chaincode stub interface for accessing ledger operations.
//   - balanceType: BalanceType - The type of balance for which the index is being created.
//
// Returns:
//   - completed: bool - True if the indexing has been completed, false if more batches remain.
//   - offset: int - The offset (position) of the last processed item in the entire dataset.
//   - err: error - An error if the index creation fails, otherwise nil.
func CreateIndex(
	stub shim.ChaincodeStubInterface,
	balanceType BalanceType,
) (bool, int, error) {
	// Create balance-type specific key for last indexed key
	balanceTypeLastIndexKey := IndexLastKeyPrefix + balanceType.String()

	// Check if indexing was already completed for this balance type
	flagCompositeKey, err := indexCreatedFlagCompositeKey(stub, balanceType)
	if err != nil {
		return false, 0, err
	}

	flagBytes, err := stub.GetState(flagCompositeKey)
	if err != nil {
		return false, 0, err
	}

	// If indexing is already completed, we can't determine the offset
	// since we're not iterating through the data anymore
	if flagBytes != nil && string(flagBytes) == "true" {
		return true, 0, nil
	}

	// Get the last indexed key to resume indexing from that point
	lastIndexedKeyBytes, err := stub.GetState(balanceTypeLastIndexKey)
	if err != nil {
		return false, 0, err
	}

	lastIndexedKey := string(lastIndexedKeyBytes)

	// Initialize the iterator
	var stateIterator shim.StateQueryIteratorInterface
	stateIterator, err = stub.GetStateByPartialCompositeKey(
		balanceType.String(),
		[]string{},
	)

	if err != nil {
		return false, 0, err
	}
	defer stateIterator.Close()

	// Counter for total items seen so far (the offset)
	offset := 0

	// If we have a last indexed key, we need to skip until we find it
	if lastIndexedKey != "" {
		for stateIterator.HasNext() {
			result, err := stateIterator.Next()
			if err != nil {
				stateIterator.Close()
				return false, offset, err
			}

			offset++

			if result.GetKey() == lastIndexedKey {
				break
			}
		}
	}

	// Process items in batches of 'limit'
	processedInBatch := 0
	lastProcessedKey := lastIndexedKey

	for stateIterator.HasNext() && processedInBatch < limit {
		result, err := stateIterator.Next()
		if err != nil {
			return false, offset, err
		}

		offset++
		currentKey := result.GetKey()
		lastProcessedKey = currentKey

		_, components, err := stub.SplitCompositeKey(currentKey)
		if err != nil {
			return false, offset, err
		}

		if len(components) < 2 {
			continue
		}

		address := components[0]
		token := components[1]
		balance := result.GetValue()

		inverseCompositeKey, err := stub.CreateCompositeKey(
			InverseBalanceObjectType,
			[]string{balanceType.String(), token, address},
		)
		if err != nil {
			return false, offset, err
		}

		if err = stub.PutState(inverseCompositeKey, balance); err != nil {
			return false, offset, err
		}

		processedInBatch++
	}

	// Update only the last indexed key in the ledger
	if err = stub.PutState(balanceTypeLastIndexKey, []byte(lastProcessedKey)); err != nil {
		return false, offset, err
	}

	// Check if we've finished processing all items
	isCompleted := !stateIterator.HasNext()

	// If completed, set the flag
	if isCompleted {
		if err = stub.PutState(flagCompositeKey, []byte("true")); err != nil {
			return false, offset, err
		}
	}

	return isCompleted, offset, nil
}

// HasIndexCreatedFlag checks if the given balance type has an index.
//
// Parameters:
//   - stub: shim.ChaincodeStubInterface
//   - balanceType: BalanceType
//
// Returns:
//   - bool: true if index exists, false otherwise
//   - error: error if any
func HasIndexCreatedFlag(
	stub shim.ChaincodeStubInterface,
	balanceType BalanceType,
) (bool, error) {
	flagCompositeKey, err := indexCreatedFlagCompositeKey(stub, balanceType)
	if err != nil {
		return false, err
	}

	flagBytes, err := stub.GetState(flagCompositeKey)
	if err != nil {
		return false, err
	}

	return len(flagBytes) > 0, nil
}

func indexCreatedFlagCompositeKey(
	stub shim.ChaincodeStubInterface,
	balanceType BalanceType,
) (string, error) {
	return stub.CreateCompositeKey(
		IndexCreatedKey,
		[]string{balanceType.String()},
	)
}
