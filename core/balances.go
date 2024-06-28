package core

import (
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
)

func (bc *BaseContract) tokenBalanceAdd(
	address *types.Address,
	amount *big.Int,
	token string,
) error {
	parts := strings.Split(token, "_")

	tokenName := ""
	if len(parts) > 1 {
		tokenName = parts[len(parts)-1]
	}

	return balance.Add(bc.stub, balance.BalanceTypeToken, address.String(), tokenName, &amount.Int)
}

func (bc *BaseContract) IndustrialBalanceGet(address *types.Address) (map[string]string, error) {
	tokens, err := balance.ListBalancesByAddress(
		bc.stub,
		balance.BalanceTypeToken,
		address.String(),
	)
	if err != nil {
		return nil, err
	}

	return tokensToMap(tokens), nil
}

func (bc *BaseContract) IndustrialBalanceTransfer(
	token string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol()+"_"+token, from, to, amount, reason)
	}

	return balance.Move(
		bc.stub,
		balance.BalanceTypeToken,
		from.String(),
		balance.BalanceTypeToken,
		to.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) IndustrialBalanceAdd(
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(
			bc.config.GetSymbol()+"_"+token,
			&types.Address{},
			address,
			amount,
			reason,
		)
	}

	return balance.Add(bc.stub, balance.BalanceTypeToken, address.String(), token, &amount.Int)
}

func (bc *BaseContract) IndustrialBalanceSub(
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(
			bc.config.GetSymbol()+"_"+token,
			address,
			&types.Address{},
			amount,
			reason,
		)
	}

	return balance.Sub(bc.stub, balance.BalanceTypeToken, address.String(), token, &amount.Int)
}

func (bc *BaseContract) TokenBalanceTransfer(
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol(), from, to, amount, reason)
	}

	return balance.Move(
		bc.stub,
		balance.BalanceTypeToken,
		from.String(),
		balance.BalanceTypeToken,
		to.String(),
		"",
		&amount.Int,
	)
}

func (bc *BaseContract) AllowedBalanceTransfer(
	token string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(token, from, to, amount, reason)
	}

	return balance.Move(
		bc.stub,
		balance.BalanceTypeAllowed,
		from.String(),
		balance.BalanceTypeAllowed,
		to.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) TokenBalanceGet(address *types.Address) (*big.Int, error) {
	balance, err := balance.Get(bc.stub, balance.BalanceTypeToken, address.String(), "")

	return new(big.Int).SetBytes(balance.Bytes()), err
}

func (bc *BaseContract) TokenBalanceAdd(
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol(), &types.Address{}, address, amount, reason)
	}

	return balance.Add(bc.stub, balance.BalanceTypeToken, address.String(), "", &amount.Int)
}

// TokenBalanceAddWithTicker adds a specified amount of tokens to an account's balance
// while recording the transaction in the ledger.
//
// Parameters:
// - address: The address of the account to add tokens to.
// - amount: The amount of tokens to add.
// - ticker: The token ticker symbol, e.g., OTF, FIAT, CURUSD, FRA_<barID>, BA_<barID>.
// - reason: The reason for adding tokens.
//
// Returns:
// - An error if the operation fails.
func (bc *BaseContract) TokenBalanceAddWithTicker(
	address *types.Address,
	amount *big.Int,
	ticker string,
	reason string,
) error {
	token, separator := "", ""
	parts := strings.Split(ticker, "_")

	// If ticker consists of multiple parts separated by '_', it indicates
	// internal subdivision into groups, bars, etc, so the balance should be accounted
	// separately for each subdivision.
	if len(parts) > 1 {
		separator = "_"
		token = parts[len(parts)-1]
	}
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol()+separator+token, address, &types.Address{}, amount, reason)
	}
	if err := balance.Add(bc.stub, balance.BalanceTypeToken, address.String(), token, &amount.Int); err != nil {
		return fmt.Errorf("failed to add token balance: %s", err.Error())
	}

	return nil
}

func (bc *BaseContract) TokenBalanceSub(
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol(), address, &types.Address{}, amount, reason)
	}

	return balance.Sub(bc.stub, balance.BalanceTypeToken, address.String(), "", &amount.Int)
}

// TokenBalanceSubWithTicker subtracts a specified amount of tokens from an account's balance
// with accounting for different tickers. It records the transaction in the ledger.
//
// Parameters:
// - address: The address of the account to subtract tokens from.
// - amount: The amount of tokens to subtract.
// - ticker: The token ticker symbol, e.g., OTF, FIAT, CURUSD, FRA_<barID>, BA_<barID>.
// - reason: The reason for subtracting tokens.
//
// Returns:
// - An error if the operation fails.
func (bc *BaseContract) TokenBalanceSubWithTicker(
	address *types.Address,
	amount *big.Int,
	ticker string,
	reason string,
) error {
	token, separator := "", ""
	parts := strings.Split(ticker, "_")

	// If ticker consists of multiple parts separated by '_', it indicates
	// internal subdivision of tokens, so the balance should be accounted
	// separately for each subdivision.
	if len(parts) > 1 {
		separator = "_"
		token = parts[len(parts)-1]
	}
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol()+separator+token, address, &types.Address{}, amount, reason)
	}
	if err := balance.Sub(bc.stub, balance.BalanceTypeToken, address.String(), token, &amount.Int); err != nil {
		return fmt.Errorf("failed to subtract token balance: %s", err.Error())
	}

	return nil
}

func (bc *BaseContract) TokenBalanceGetLocked(address *types.Address) (*big.Int, error) {
	balance, err := balance.Get(bc.stub, balance.BalanceTypeTokenLocked, address.String(), "")

	return new(big.Int).SetBytes(balance.Bytes()), err
}

func (bc *BaseContract) TokenBalanceLock(address *types.Address, amount *big.Int) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol(), address, address, amount, "token balance lock")
	}
	return balance.Move(
		bc.stub,
		balance.BalanceTypeToken,
		address.String(),
		balance.BalanceTypeTokenLocked,
		address.String(),
		"",
		&amount.Int,
	)
}

func (bc *BaseContract) TokenBalanceUnlock(address *types.Address, amount *big.Int) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol(), address, address, amount, "token balance unlock")
	}
	return balance.Move(
		bc.stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
		balance.BalanceTypeToken,
		address.String(),
		"",
		&amount.Int,
	)
}

func (bc *BaseContract) TokenBalanceTransferLocked(
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol(), from, to, amount, reason)
	}

	return balance.Move(
		bc.stub,
		balance.BalanceTypeTokenLocked,
		from.String(),
		balance.BalanceTypeToken,
		to.String(),
		"",
		&amount.Int,
	)
}

func (bc *BaseContract) TokenBalanceBurnLocked(
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol(), address, &types.Address{}, amount, reason)
	}

	return balance.Sub(
		bc.stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
		"",
		&amount.Int,
	)
}

func (bc *BaseContract) AllowedBalanceGet(token string, address *types.Address) (*big.Int, error) {
	balance, err := balance.Get(bc.stub, balance.BalanceTypeAllowed, address.String(), token)

	return new(big.Int).SetBytes(balance.Bytes()), err
}

func (bc *BaseContract) AllowedBalanceAdd(
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(token, &types.Address{}, address, amount, reason)
	}

	return balance.Add(bc.stub, balance.BalanceTypeAllowed, address.String(), token, &amount.Int)
}

func (bc *BaseContract) AllowedBalanceSub(
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(token, address, &types.Address{}, amount, reason)
	}

	return balance.Sub(
		bc.stub,
		balance.BalanceTypeAllowed,
		address.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) AllowedIndustrialBalanceTransfer(
	from *types.Address,
	to *types.Address,
	industrialAssets []*pb.Asset,
	reason string,
) error {
	for _, industrialAsset := range industrialAssets {
		amount := new(big.Int).SetBytes(industrialAsset.GetAmount())
		if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
			stub.AddAccountingRecord(industrialAsset.GetGroup(), from, to, amount, reason)
		}

		if err := balance.Move(
			bc.stub,
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

func (bc *BaseContract) AllowedIndustrialBalanceAdd(
	address *types.Address,
	industrialAssets []*pb.Asset,
	reason string,
) error {
	for _, industrialAsset := range industrialAssets {
		amount := new(big.Int).SetBytes(industrialAsset.GetAmount())
		if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
			stub.AddAccountingRecord(
				industrialAsset.GetGroup(),
				&types.Address{},
				address,
				amount,
				reason,
			)
		}

		if err := balance.Add(
			bc.stub,
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

func (bc *BaseContract) AllowedIndustrialBalanceSub(
	address *types.Address,
	industrialAssets []*pb.Asset,
	reason string,
) error {
	for _, asset := range industrialAssets {
		amount := new(big.Int).SetBytes(asset.GetAmount())
		if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
			stub.AddAccountingRecord(asset.GetGroup(), address, &types.Address{}, amount, reason)
		}

		if err := balance.Sub(
			bc.stub,
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

func (bc *BaseContract) AllowedBalanceGetLocked(token string, address *types.Address) (*big.Int, error) {
	balanceValue, err := balance.Get(bc.stub, balance.BalanceTypeAllowedLocked, address.String(), token)
	return new(big.Int).SetBytes(balanceValue.Bytes()), err
}

func (bc *BaseContract) AllowedBalanceLock(
	token string,
	address *types.Address,
	amount *big.Int,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol(), address, address, amount, "allowed balance lock")
	}

	return balance.Move(
		bc.stub,
		balance.BalanceTypeAllowed,
		address.String(),
		balance.BalanceTypeAllowedLocked,
		address.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) AllowedBalanceUnLock(
	token string,
	address *types.Address,
	amount *big.Int,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol(), address, address, amount, "allowed balance unlock")
	}

	return balance.Move(
		bc.stub,
		balance.BalanceTypeAllowedLocked,
		address.String(),
		balance.BalanceTypeAllowed,
		address.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) AllowedBalanceTransferLocked(
	token string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(token, from, to, amount, reason)
	}

	return balance.Move(
		bc.stub,
		balance.BalanceTypeAllowedLocked,
		from.String(),
		balance.BalanceTypeAllowed,
		to.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) AllowedBalanceBurnLocked(
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(token, address, &types.Address{}, amount, reason)
	}

	return balance.Sub(
		bc.stub,
		balance.BalanceTypeAllowedLocked,
		address.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) IndustrialBalanceGetLocked(
	address *types.Address,
) (map[string]string, error) {
	tokens, err := balance.ListBalancesByAddress(
		bc.stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
	)
	if err != nil {
		return nil, err
	}

	return tokensToMap(tokens), nil
}

func (bc *BaseContract) IndustrialBalanceLock(
	token string,
	address *types.Address,
	amount *big.Int,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol()+"_"+token, address, address, amount, "industrial balance lock")
	}

	return balance.Move(
		bc.stub,
		balance.BalanceTypeToken,
		address.String(),
		balance.BalanceTypeTokenLocked,
		address.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) IndustrialBalanceUnLock(
	token string,
	address *types.Address,
	amount *big.Int,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol()+"_"+token, address, address, amount, "industrial balance unlock")
	}

	return balance.Move(
		bc.stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
		balance.BalanceTypeToken,
		address.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) IndustrialBalanceTransferLocked(
	token string,
	from *types.Address,
	to *types.Address,
	amount *big.Int,
	reason string,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(bc.config.GetSymbol()+"_"+token, from, to, amount, reason)
	}

	return balance.Move(
		bc.stub,
		balance.BalanceTypeTokenLocked,
		from.String(),
		balance.BalanceTypeToken,
		to.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) IndustrialBalanceBurnLocked(
	token string,
	address *types.Address,
	amount *big.Int,
	reason string,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*cachestub.TxCacheStub); ok {
		stub.AddAccountingRecord(
			bc.config.GetSymbol()+"_"+token,
			address,
			&types.Address{},
			amount,
			reason,
		)
	}

	return balance.Sub(
		bc.stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
		token,
		&amount.Int,
	)
}

func (bc *BaseContract) AllowedBalanceGetAll(address *types.Address) (map[string]string, error) {
	tokens, err := balance.ListBalancesByAddress(
		bc.stub,
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
