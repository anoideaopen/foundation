package balance

import (
	"errors"
	"math/big"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
)

// Error definitions for balance operations.
var (
	ErrAmountMustBeNonNegative = errors.New("amount must be non-negative")
	ErrInsufficientBalance     = errors.New("insufficient balance")
)

// Add adds the given amount to the balance for the specified address and token, if the amount is greater than zero.
//
// Parameters:
//   - stub: shim.ChaincodeStubInterface - The chaincode stub interface for accessing ledger operations.
//   - balanceType: BalanceType - The type of balance to update, which determines the state key's prefix.
//   - address: string - The address associated with the balance.
//   - token: string - The token identifier. If empty, the balance associated with the address alone is updated.
//   - amount: *big.Int - The amount to add to the balance associated with the address and token.
//
// Returns:
//   - error - An error if the addition fails, otherwise nil.
func Add(
	stub shim.ChaincodeStubInterface,
	balanceType BalanceType,
	address string,
	token string,
	amount *big.Int,
) error {
	if amount.Sign() < 0 {
		return ErrAmountMustBeNonNegative
	}

	currentBalance, err := Get(stub, balanceType, address, token)
	if err != nil {
		return err
	}

	newBalance := new(big.Int).Add(currentBalance, amount)
	return Put(stub, balanceType, address, token, newBalance)
}

// Sub subtracts the given amount from the balance for the specified address and token, if the amount is greater than zero.
//
// Parameters:
//   - stub: shim.ChaincodeStubInterface - The chaincode stub interface for accessing ledger operations.
//   - balanceType: BalanceType - The type of balance to update, which determines the state key's prefix.
//   - address: string - The address associated with the balance.
//   - token: string - The token identifier. If empty, the balance associated with the address alone is updated.
//   - amount: *big.Int - The amount to subtract from the balance associated with the address and token.
//
// Returns:
//   - error - An error if the subtraction fails, otherwise nil.
func Sub(
	stub shim.ChaincodeStubInterface,
	balanceType BalanceType,
	address string,
	token string,
	amount *big.Int,
) error {
	if amount.Sign() < 0 {
		return ErrAmountMustBeNonNegative
	}

	currentBalance, err := Get(stub, balanceType, address, token)
	if err != nil {
		return err
	}

	if currentBalance.Cmp(amount) < 0 {
		return ErrInsufficientBalance
	}

	newBalance := new(big.Int).Sub(currentBalance, amount)
	return Put(stub, balanceType, address, token, newBalance)
}

// Move moves the given amount from the balance of one address and balance type to the balance of another address and balance type.
//
// Parameters:
//   - stub: shim.ChaincodeStubInterface - The chaincode stub interface for accessing ledger operations.
//   - sourceBalanceType: BalanceType - The type of balance from which the amount will be subtracted.
//   - sourceAddress: string - The address from which the amount will be subtracted.
//   - destBalanceType: BalanceType - The type of balance to which the amount will be added.
//   - destAddress: string - The address to which the amount will be added.
//   - token: string - The token identifier. If empty, the operation is performed on the balances associated with the addresses alone.
//   - amount: *big.Int - The amount to transfer from the source balance and address to the destination balance and address.
//
// Returns:
//   - error - An error if the transfer fails, otherwise nil.
func Move(
	stub shim.ChaincodeStubInterface,
	sourceBalanceType BalanceType,
	sourceAddress string,
	destBalanceType BalanceType,
	destAddress string,
	token string,
	amount *big.Int,
) error {
	if err := Sub(stub, sourceBalanceType, sourceAddress, token, amount); err != nil {
		return err
	}

	return Add(stub, destBalanceType, destAddress, token, amount)
}
