package balance

import (
	"errors"
	"math/big"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
)

// InverseBalanceObjectType is designed for indexing the inverse balance values to retrieve
// a list of token owners.
const InverseBalanceObjectType = "inverse_balance"

var ErrAddressMustNotBeEmpty = errors.New("address must not be empty")

// Get retrieves the balance value for the given address and token, constructing the appropriate composite key.
//
// Parameters:
//   - stub: shim.ChaincodeStubInterface - The chaincode stub interface for accessing ledger operations.
//   - balanceType: BalanceType - The type of balance to retrieve, which determines the state key's prefix.
//   - address: string - The address associated with the balance.
//   - token: string - The token identifier. If empty, the balance associated with the address alone is retrieved.
//
// Returns:
//   - *big.Int - The balance value associated with the composite key.
//   - error - An error if the retrieval fails, otherwise nil.
func Get(
	stub shim.ChaincodeStubInterface,
	balanceType BalanceType,
	address string,
	token string,
) (*big.Int, error) {
	if address == "" {
		return nil, ErrAddressMustNotBeEmpty
	}

	// Construct the composite key based on the address and, if provided, the token.
	compositeKeyAttributes := []string{address}
	if token != "" {
		compositeKeyAttributes = append(compositeKeyAttributes, token)
	}

	compositeKey, err := stub.CreateCompositeKey(balanceType.String(), compositeKeyAttributes)
	if err != nil {
		return nil, err
	}

	// Retrieve the balance from the ledger using the composite key.
	balanceBytes, err := stub.GetState(compositeKey)
	if err != nil {
		return nil, err
	}

	// Convert the balance from bytes to *big.Int and return.
	balance := new(big.Int).SetBytes(balanceBytes)
	return balance, nil
}

// Put stores the balance for a given address and token into the ledger.
//
// Parameters:
//   - stub: shim.ChaincodeStubInterface - The chaincode stub interface for accessing ledger operations.
//   - balanceType: BalanceType - The type of balance to store, which determines the state key's prefix.
//   - address: string - The address associated with the balance.
//   - token: string - The token identifier. If empty, the balance associated with the address alone is stored.
//   - value: *big.Int - The balance value to store associated with the address and token.
//
// Returns:
//   - error - An error if the storage fails, otherwise nil.
func Put(
	stub shim.ChaincodeStubInterface,
	balanceType BalanceType,
	address string,
	token string,
	value *big.Int,
) error {
	if address == "" {
		return ErrAddressMustNotBeEmpty
	}

	// Create the primary composite key for the balance entry.
	primaryAttributes := []string{address}
	if token != "" {
		primaryAttributes = append(primaryAttributes, token)
	}
	primaryCompositeKey, err := stub.CreateCompositeKey(balanceType.String(), primaryAttributes)
	if err != nil {
		return err
	}

	// Store the balance using the primary composite key.
	if err := stub.PutState(primaryCompositeKey, value.Bytes()); err != nil {
		return err
	}

	// If the token is not specified, there's no need to create an inverse index.
	if token == "" {
		return nil
	}

	// Create the inverse composite key for the balance entry.
	inverseCompositeKey, err := stub.CreateCompositeKey(
		InverseBalanceObjectType,
		[]string{balanceType.String(), token, address},
	)
	if err != nil {
		return err
	}

	// Store the balance using the inverse composite key.
	return stub.PutState(inverseCompositeKey, value.Bytes())
}
