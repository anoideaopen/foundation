package balance

import (
	"math/big"

	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// TokenBalance represents a balance entry with a token identifier and its associated value.
type TokenBalance struct {
	Address string
	Token   string
	Balance *big.Int
}

// ListBalancesByAddress fetches all balance entries associated with the given address.
//
// Parameters:
//   - stub: shim.ChaincodeStubInterface - Interface for accessing ledger operations.
//   - balanceType: BalanceType - The type of balance to retrieve, determining the state key's prefix.
//   - address: string - The address whose balances are to be fetched.
//
// Returns:
//   - []TokenBalance - A slice of TokenBalance structs representing all balances associated with the address.
//   - error - An error if the retrieval fails, otherwise nil.
func ListBalancesByAddress(
	stub shim.ChaincodeStubInterface,
	balanceType BalanceType,
	address string,
) ([]TokenBalance, error) {
	stateIterator, err := stub.GetStateByPartialCompositeKey(
		balanceType.String(),
		[]string{address},
	)
	if err != nil {
		return nil, err
	}
	defer stateIterator.Close()

	var balances []TokenBalance
	for stateIterator.HasNext() {
		response, err := stateIterator.Next()
		if err != nil {
			return nil, err
		}

		_, components, err := stub.SplitCompositeKey(response.GetKey())
		if err != nil {
			return nil, err
		}

		if len(components) < 2 {
			continue
		}

		balances = append(balances, TokenBalance{
			Address: components[0],
			Token:   components[1],
			Balance: new(big.Int).SetBytes(response.GetValue()),
		})
	}

	return balances, nil
}

// ListOwnersByToken fetches all owners and their balances for a specific token.
//
// Parameters:
//   - stub: shim.ChaincodeStubInterface - Interface for accessing ledger operations.
//   - balanceType: BalanceType - The type of balance to retrieve, determining the state key's prefix.
//   - token: string - The token identifier whose owners are to be fetched.
//
// Returns:
//   - []TokenBalance - A slice of TokenBalance structs representing all owners and their balances for the token.
//   - error - An error if the retrieval fails, otherwise nil.
func ListOwnersByToken(
	stub shim.ChaincodeStubInterface,
	balanceType BalanceType,
	token string,
) ([]TokenBalance, error) {
	stateIterator, err := stub.GetStateByPartialCompositeKey(
		InverseBalanceObjectType,
		[]string{balanceType.String(), token},
	)
	if err != nil {
		return nil, err
	}
	defer stateIterator.Close()

	var owners []TokenBalance
	for stateIterator.HasNext() {
		response, err := stateIterator.Next()
		if err != nil {
			return nil, err
		}

		_, components, err := stub.SplitCompositeKey(response.GetKey())
		if err != nil {
			return nil, err
		}

		if len(components) < 3 {
			continue
		}

		owners = append(owners, TokenBalance{
			Token:   components[1],
			Address: components[2],
			Balance: new(big.Int).SetBytes(response.GetValue()),
		})
	}

	return owners, nil
}
