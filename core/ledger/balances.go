package ledger

import (
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

func TokenBalanceAdd(
	stub shim.ChaincodeStubInterface,
	symbol string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(symbol, &types.Address{}, address, amount, reason)
	}

	return balance.Add(stub, balance.BalanceTypeToken, address.String(), "", &amount.Int)
}

func IndustrialBalanceGet(
	stub shim.ChaincodeStubInterface,
	address *types.Address,
) (map[string]string, error) {
	tokens, err := balance.ListBalancesByAddress(
		stub,
		balance.BalanceTypeToken,
		address.String(),
	)
	if err != nil {
		return nil, err
	}
	return tokensToMap(tokens), nil
}

func IndustrialBalanceTransfer(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(symbol+"_"+token, from, to, amount, reason)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeToken,
		from.String(),
		balance.BalanceTypeToken,
		to.String(),
		token,
		&amount.Int,
	)
}

func IndustrialBalanceAdd(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol+"_"+token,
			&types.Address{},
			address,
			amount,
			reason,
		)
	}
	return balance.Add(stub, balance.BalanceTypeToken, address.String(), token, &amount.Int)
}

func IndustrialBalanceSub(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol+"_"+token,
			address,
			&types.Address{},
			amount,
			reason,
		)
	}
	return balance.Sub(stub, balance.BalanceTypeToken, address.String(), token, &amount.Int)
}

func TokenBalanceTransfer(
	stub shim.ChaincodeStubInterface,
	symbol string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(symbol, from, to, amount, reason)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeToken,
		from.String(),
		balance.BalanceTypeToken,
		to.String(),
		"",
		&amount.Int,
	)
}

func AllowedBalanceTransfer(
	stub shim.ChaincodeStubInterface,
	token string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(token, from, to, amount, reason)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeAllowed,
		from.String(),
		balance.BalanceTypeAllowed,
		to.String(),
		token,
		&amount.Int,
	)
}

func TokenBalanceGet(stub shim.ChaincodeStubInterface, address *types.Address) (*big.Int, error) {
	balance, err := balance.Get(stub, balance.BalanceTypeToken, address.String(), "")
	return new(big.Int).SetBytes(balance.Bytes()), err
}

func TokenBalanceAddWithReason(
	stub shim.ChaincodeStubInterface,
	symbol string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(symbol, &types.Address{}, address, amount, reason)
	}
	return balance.Add(stub, balance.BalanceTypeToken, address.String(), "", &amount.Int)
}

func TokenBalanceAddWithTicker(
	stub shim.ChaincodeStubInterface,
	symbol string,
	address *types.Address,
	amount *big.Int,
	ticker string,
	reason string,
) error {
	token, separator := "", ""
	parts := strings.Split(ticker, "_")
	if len(parts) > 1 {
		separator = "_"
		token = parts[len(parts)-1]
	}
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol+separator+token,
			address,
			&types.Address{},
			amount,
			reason,
		)
	}
	if err := balance.Add(stub, balance.BalanceTypeToken, address.String(), token, &amount.Int); err != nil {
		return fmt.Errorf("failed to add token balance: %s", err.Error())
	}
	return nil
}

func TokenBalanceSub(
	stub shim.ChaincodeStubInterface,
	symbol string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(symbol, address, &types.Address{}, amount, reason)
	}
	return balance.Sub(stub, balance.BalanceTypeToken, address.String(), "", &amount.Int)
}

func TokenBalanceSubWithTicker(
	stub shim.ChaincodeStubInterface,
	symbol string,
	address *types.Address,
	amount *big.Int,
	ticker string,
	reason string,
) error {
	token, separator := "", ""
	parts := strings.Split(ticker, "_")
	if len(parts) > 1 {
		separator = "_"
		token = parts[len(parts)-1]
	}
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol+separator+token,
			address,
			&types.Address{},
			amount,
			reason,
		)
	}
	if err := balance.Sub(stub, balance.BalanceTypeToken, address.String(), token, &amount.Int); err != nil {
		return fmt.Errorf("failed to subtract token balance: %s", err.Error())
	}
	return nil
}

func TokenBalanceGetLocked(
	stub shim.ChaincodeStubInterface,
	address *types.Address,
) (*big.Int, error) {
	balance, err := balance.Get(stub, balance.BalanceTypeTokenLocked, address.String(), "")
	return new(big.Int).SetBytes(balance.Bytes()), err
}

func TokenBalanceLock(
	stub shim.ChaincodeStubInterface,
	symbol string,
	address *types.Address,
	amount *big.Int,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(symbol, address, address, amount, "token balance lock")
	}
	return balance.Move(
		stub,
		balance.BalanceTypeToken,
		address.String(),
		balance.BalanceTypeTokenLocked,
		address.String(),
		"",
		&amount.Int,
	)
}

func TokenBalanceUnlock(
	stub shim.ChaincodeStubInterface,
	symbol string,
	address *types.Address,
	amount *big.Int,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol,
			address,
			address,
			amount,
			"token balance unlock",
		)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
		balance.BalanceTypeToken,
		address.String(),
		"",
		&amount.Int,
	)
}

func TokenBalanceTransferLocked(
	stub shim.ChaincodeStubInterface,
	symbol string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(symbol, from, to, amount, reason)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeTokenLocked,
		from.String(),
		balance.BalanceTypeToken,
		to.String(),
		"",
		&amount.Int,
	)
}

func TokenBalanceBurnLocked(
	stub shim.ChaincodeStubInterface,
	symbol string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(symbol, address, &types.Address{}, amount, reason)
	}
	return balance.Sub(
		stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
		"",
		&amount.Int,
	)
}

func AllowedBalanceGet(
	stub shim.ChaincodeStubInterface,
	token string,
	address *types.Address,
) (*big.Int, error) {
	balance, err := balance.Get(stub, balance.BalanceTypeAllowed, address.String(), token)
	return new(big.Int).SetBytes(balance.Bytes()), err
}

func AllowedBalanceAdd(
	stub shim.ChaincodeStubInterface,
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(token, &types.Address{}, address, amount, reason)
	}
	return balance.Add(stub, balance.BalanceTypeAllowed, address.String(), token, &amount.Int)
}

func AllowedBalanceSub(
	stub shim.ChaincodeStubInterface,
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(token, address, &types.Address{}, amount, reason)
	}
	return balance.Sub(
		stub,
		balance.BalanceTypeAllowed,
		address.String(),
		token,
		&amount.Int,
	)
}

func AllowedIndustrialBalanceTransfer(
	stub shim.ChaincodeStubInterface,
	from *types.Address,
	to *types.Address,
	industrialAssets []*pb.Asset,
	reason string,
) error {
	for _, industrialAsset := range industrialAssets {
		amount := new(big.Int).SetBytes(industrialAsset.GetAmount())
		if stub, ok := stub.(Accounting); ok {
			stub.AddAccountingRecord(industrialAsset.GetGroup(), from, to, amount, reason)
		}
		if err := balance.Move(
			stub,
			balance.BalanceTypeAllowed,
			from.String(),
			balance.BalanceTypeAllowed,
			to.String(),
			industrialAsset.GetGroup(),
			&amount.Int,
		); err != nil {
			return err
		}
	}
	return nil
}

func AllowedIndustrialBalanceAdd(
	stub shim.ChaincodeStubInterface,
	address *types.Address,
	industrialAssets []*pb.Asset,
	reason string,
) error {
	for _, industrialAsset := range industrialAssets {
		amount := new(big.Int).SetBytes(industrialAsset.GetAmount())
		if stub, ok := stub.(Accounting); ok {
			stub.AddAccountingRecord(
				industrialAsset.GetGroup(),
				&types.Address{},
				address,
				amount,
				reason,
			)
		}
		if err := balance.Add(
			stub,
			balance.BalanceTypeAllowed,
			address.String(),
			industrialAsset.GetGroup(),
			&amount.Int,
		); err != nil {
			return err
		}
	}
	return nil
}

func AllowedIndustrialBalanceSub(
	stub shim.ChaincodeStubInterface,
	address *types.Address,
	industrialAssets []*pb.Asset,
	reason string,
) error {
	for _, asset := range industrialAssets {
		amount := new(big.Int).SetBytes(asset.GetAmount())
		if stub, ok := stub.(Accounting); ok {
			stub.AddAccountingRecord(asset.GetGroup(), address, &types.Address{}, amount, reason)
		}
		if err := balance.Sub(
			stub,
			balance.BalanceTypeAllowed,
			address.String(),
			asset.GetGroup(),
			&amount.Int,
		); err != nil {
			return err
		}
	}
	return nil
}

func AllowedBalanceGetLocked(
	stub shim.ChaincodeStubInterface,
	token string,
	address *types.Address,
) (*big.Int, error) {
	balanceValue, err := balance.Get(
		stub,
		balance.BalanceTypeAllowedLocked,
		address.String(),
		token,
	)
	return new(big.Int).SetBytes(balanceValue.Bytes()), err
}

func AllowedBalanceLock(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol,
			address,
			address,
			amount,
			"allowed balance lock",
		)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeAllowed,
		address.String(),
		balance.BalanceTypeAllowedLocked,
		address.String(),
		token,
		&amount.Int,
	)
}

func AllowedBalanceUnlock(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol,
			address,
			address,
			amount,
			"allowed balance unlock",
		)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeAllowedLocked,
		address.String(),
		balance.BalanceTypeAllowed,
		address.String(),
		token,
		&amount.Int,
	)
}

func AllowedBalanceTransferLocked(
	stub shim.ChaincodeStubInterface,
	token string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(token, from, to, amount, reason)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeAllowedLocked,
		from.String(),
		balance.BalanceTypeAllowed,
		to.String(),
		token,
		&amount.Int,
	)
}

func AllowedBalanceBurnLocked(
	stub shim.ChaincodeStubInterface,
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(token, address, &types.Address{}, amount, reason)
	}
	return balance.Sub(
		stub,
		balance.BalanceTypeAllowedLocked,
		address.String(),
		token,
		&amount.Int,
	)
}

func IndustrialBalanceGetLocked(
	stub shim.ChaincodeStubInterface,
	address *types.Address,
) (map[string]string, error) {
	tokens, err := balance.ListBalancesByAddress(
		stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
	)
	if err != nil {
		return nil, err
	}
	return tokensToMap(tokens), nil
}

func IndustrialBalanceLock(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol+"_"+token,
			address,
			address,
			amount,
			"industrial balance lock",
		)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeToken,
		address.String(),
		balance.BalanceTypeTokenLocked,
		address.String(),
		token,
		&amount.Int,
	)
}

func IndustrialBalanceUnlock(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol+"_"+token,
			address,
			address,
			amount,
			"industrial balance unlock",
		)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
		balance.BalanceTypeToken,
		address.String(),
		token,
		&amount.Int,
	)
}

func IndustrialBalanceTransferLocked(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(symbol+"_"+token, from, to, amount, reason)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeTokenLocked,
		from.String(),
		balance.BalanceTypeToken,
		to.String(),
		token,
		&amount.Int,
	)
}

func IndustrialBalanceBurnLocked(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol+"_"+token,
			address,
			&types.Address{},
			amount,
			reason,
		)
	}
	return balance.Sub(
		stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
		token,
		&amount.Int,
	)
}

func AllowedBalanceGetAll(
	stub shim.ChaincodeStubInterface,
	address *types.Address,
) (map[string]string, error) {
	tokens, err := balance.ListBalancesByAddress(
		stub,
		balance.BalanceTypeAllowed,
		address.String(),
	)
	if err != nil {
		return nil, err
	}
	return tokensToMap(tokens), nil
}

func tokensToMap(tokens []balance.TokenBalance) map[string]string {
	balances := make(map[string]string)
	for _, item := range tokens {
		balances[item.Token] = item.Balance.String()
	}
	return balances
}
