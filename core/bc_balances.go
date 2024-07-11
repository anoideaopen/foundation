package core

import (
	"github.com/anoideaopen/foundation/core/ledger"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
)

func (bc *BaseContract) TokenBalanceAdd(address *types.Address, amount *big.Int, token string) error {
	return ledger.TokenBalanceAdd(bc.GetStub(), address, amount, token)
}

func (bc *BaseContract) IndustrialBalanceGet(address *types.Address) (map[string]string, error) {
	return ledger.IndustrialBalanceGet(bc.GetStub(), address)
}

func (bc *BaseContract) IndustrialBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.IndustrialBalanceTransfer(bc.GetStub(), bc.ContractConfig().GetSymbol(), token, from, to, amount, reason)
}

func (bc *BaseContract) IndustrialBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.IndustrialBalanceAdd(bc.GetStub(), bc.ContractConfig().GetSymbol(), token, address, amount, reason)
}

func (bc *BaseContract) IndustrialBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.IndustrialBalanceSub(bc.GetStub(), bc.ContractConfig().GetSymbol(), token, address, amount, reason)
}

func (bc *BaseContract) TokenBalanceTransfer(from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.TokenBalanceTransfer(bc.GetStub(), bc.ContractConfig().GetSymbol(), from, to, amount, reason)
}

func (bc *BaseContract) AllowedBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.AllowedBalanceTransfer(bc.GetStub(), token, from, to, amount, reason)
}

func (bc *BaseContract) TokenBalanceGet(address *types.Address) (*big.Int, error) {
	return ledger.TokenBalanceGet(bc.GetStub(), address)
}

func (bc *BaseContract) TokenBalanceAddWithReason(address *types.Address, amount *big.Int, reason string) error {
	return ledger.TokenBalanceAddWithReason(bc.GetStub(), bc.ContractConfig().GetSymbol(), address, amount, reason)
}

func (bc *BaseContract) TokenBalanceAddWithTicker(address *types.Address, amount *big.Int, ticker string, reason string) error {
	return ledger.TokenBalanceAddWithTicker(bc.GetStub(), bc.ContractConfig().GetSymbol(), address, amount, ticker, reason)
}

func (bc *BaseContract) TokenBalanceSub(address *types.Address, amount *big.Int, reason string) error {
	return ledger.TokenBalanceSub(bc.GetStub(), bc.ContractConfig().GetSymbol(), address, amount, reason)
}

func (bc *BaseContract) TokenBalanceSubWithTicker(address *types.Address, amount *big.Int, ticker string, reason string) error {
	return ledger.TokenBalanceSubWithTicker(bc.GetStub(), bc.ContractConfig().GetSymbol(), address, amount, ticker, reason)
}

func (bc *BaseContract) TokenBalanceGetLocked(address *types.Address) (*big.Int, error) {
	return ledger.TokenBalanceGetLocked(bc.GetStub(), address)
}

func (bc *BaseContract) TokenBalanceLock(address *types.Address, amount *big.Int) error {
	return ledger.TokenBalanceLock(bc.GetStub(), bc.ContractConfig().GetSymbol(), address, amount)
}

func (bc *BaseContract) TokenBalanceUnlock(address *types.Address, amount *big.Int) error {
	return ledger.TokenBalanceUnlock(bc.GetStub(), bc.ContractConfig().GetSymbol(), address, amount)
}

func (bc *BaseContract) TokenBalanceTransferLocked(from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.TokenBalanceTransferLocked(bc.GetStub(), bc.ContractConfig().GetSymbol(), from, to, amount, reason)
}

func (bc *BaseContract) TokenBalanceBurnLocked(address *types.Address, amount *big.Int, reason string) error {
	return ledger.TokenBalanceBurnLocked(bc.GetStub(), bc.ContractConfig().GetSymbol(), address, amount, reason)
}

func (bc *BaseContract) AllowedBalanceGet(token string, address *types.Address) (*big.Int, error) {
	return ledger.AllowedBalanceGet(bc.GetStub(), token, address)
}

func (bc *BaseContract) AllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.AllowedBalanceAdd(bc.GetStub(), token, address, amount, reason)
}

func (bc *BaseContract) AllowedBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.AllowedBalanceSub(bc.GetStub(), token, address, amount, reason)
}

func (bc *BaseContract) AllowedIndustrialBalanceTransfer(from *types.Address, to *types.Address, industrialAssets []*pb.Asset, reason string) error {
	return ledger.AllowedIndustrialBalanceTransfer(bc.GetStub(), from, to, industrialAssets, reason)
}

func (bc *BaseContract) AllowedIndustrialBalanceAdd(address *types.Address, industrialAssets []*pb.Asset, reason string) error {
	return ledger.AllowedIndustrialBalanceAdd(bc.GetStub(), address, industrialAssets, reason)
}

func (bc *BaseContract) AllowedIndustrialBalanceSub(address *types.Address, industrialAssets []*pb.Asset, reason string) error {
	return ledger.AllowedIndustrialBalanceSub(bc.GetStub(), address, industrialAssets, reason)
}

func (bc *BaseContract) AllowedBalanceGetLocked(token string, address *types.Address) (*big.Int, error) {
	return ledger.AllowedBalanceGetLocked(bc.GetStub(), token, address)
}

func (bc *BaseContract) AllowedBalanceLock(token string, address *types.Address, amount *big.Int) error {
	return ledger.AllowedBalanceLock(bc.GetStub(), bc.ContractConfig().GetSymbol(), token, address, amount)
}

func (bc *BaseContract) AllowedBalanceUnLock(token string, address *types.Address, amount *big.Int) error {
	return ledger.AllowedBalanceUnlock(bc.GetStub(), bc.ContractConfig().GetSymbol(), token, address, amount)
}

func (bc *BaseContract) AllowedBalanceTransferLocked(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.AllowedBalanceTransferLocked(bc.GetStub(), token, from, to, amount, reason)
}

func (bc *BaseContract) AllowedBalanceBurnLocked(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.AllowedBalanceBurnLocked(bc.GetStub(), token, address, amount, reason)
}

func (bc *BaseContract) IndustrialBalanceGetLocked(address *types.Address) (map[string]string, error) {
	return ledger.IndustrialBalanceGetLocked(bc.GetStub(), address)
}

func (bc *BaseContract) IndustrialBalanceLock(token string, address *types.Address, amount *big.Int) error {
	return ledger.IndustrialBalanceLock(bc.GetStub(), bc.ContractConfig().GetSymbol(), token, address, amount)
}

func (bc *BaseContract) IndustrialBalanceUnLock(token string, address *types.Address, amount *big.Int) error {
	return ledger.IndustrialBalanceUnlock(bc.GetStub(), bc.ContractConfig().GetSymbol(), token, address, amount)
}

func (bc *BaseContract) IndustrialBalanceTransferLocked(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.IndustrialBalanceTransferLocked(bc.GetStub(), bc.ContractConfig().GetSymbol(), token, from, to, amount, reason)
}

func (bc *BaseContract) IndustrialBalanceBurnLocked(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.IndustrialBalanceBurnLocked(bc.GetStub(), bc.ContractConfig().GetSymbol(), token, address, amount, reason)
}

func (bc *BaseContract) AllowedBalanceGetAll(address *types.Address) (map[string]string, error) {
	return ledger.AllowedBalanceGetAll(bc.GetStub(), address)
}
