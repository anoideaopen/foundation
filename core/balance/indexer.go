package balance

import "github.com/hyperledger/fabric-chaincode-go/shim"

// IMPORTANT: THE INDEXER CAN BE USED AS A TOOL FOR MIGRATING EXISTING
// TOKENS. DETAILS IN README.md.

// IndexCreatedKey is the key used to store the index creation flag.
const IndexCreatedKey = "balance_index_created"

// CreateIndex builds an index for states matching the specified balance type.
//
// Parameters:
//   - stub: shim.ChaincodeStubInterface - The chaincode stub interface for accessing ledger operations.
//   - balanceType: BalanceType - The type of balance for which the index is being created.
//
// Returns:
//   - err: error - An error if the index creation fails, otherwise nil.
func CreateIndex(
	stub shim.ChaincodeStubInterface,
	balanceType BalanceType,
) error {
	// Retrieve an iterator for the states.
	stateIterator, err := stub.GetStateByPartialCompositeKey(
		balanceType.String(),
		[]string{},
	)
	if err != nil {
		return err
	}
	defer stateIterator.Close()

	for stateIterator.HasNext() {
		result, err := stateIterator.Next()
		if err != nil {
			return err
		}

		_, components, err := stub.SplitCompositeKey(result.GetKey())
		if err != nil {
			return err
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
			return err
		}

		if err := stub.PutState(inverseCompositeKey, balance); err != nil {
			return err
		}
	}

	flagCompositeKey, err := indexCreatedFlagCompositeKey(stub, balanceType)
	if err != nil {
		return err
	}

	return stub.PutState(flagCompositeKey, []byte("true"))
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
