package core

import (
	"github.com/anoideaopen/foundation/core/ledger"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
)

func (bc *BaseContract) TokenBalanceAdd(address *types.Address, amount *big.Int, token string) error {
	return ledger.TokenBalanceAdd(bc.stub, address, amount, token)
}

func (bc *BaseContract) IndustrialBalanceGet(address *types.Address) (map[string]string, error) {
	return ledger.IndustrialBalanceGet(bc.stub, address)
}

func (bc *BaseContract) IndustrialBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.IndustrialBalanceTransfer(bc.stub, bc.config.GetSymbol(), token, from, to, amount, reason)
}

func (bc *BaseContract) IndustrialBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.IndustrialBalanceAdd(bc.stub, bc.config.GetSymbol(), token, address, amount, reason)
}

func (bc *BaseContract) IndustrialBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.IndustrialBalanceSub(bc.stub, bc.config.GetSymbol(), token, address, amount, reason)
}

func (bc *BaseContract) TokenBalanceTransfer(from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.TokenBalanceTransfer(bc.stub, bc.config.GetSymbol(), from, to, amount, reason)
}

func (bc *BaseContract) AllowedBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.AllowedBalanceTransfer(bc.stub, token, from, to, amount, reason)
}

func (bc *BaseContract) TokenBalanceGet(address *types.Address) (*big.Int, error) {
	return ledger.TokenBalanceGet(bc.stub, address)
}

func (bc *BaseContract) TokenBalanceAddWithReason(address *types.Address, amount *big.Int, reason string) error {
	return ledger.TokenBalanceAddWithReason(bc.stub, bc.config.GetSymbol(), address, amount, reason)
}

func (bc *BaseContract) TokenBalanceAddWithTicker(address *types.Address, amount *big.Int, ticker string, reason string) error {
	return ledger.TokenBalanceAddWithTicker(bc.stub, bc.config.GetSymbol(), address, amount, ticker, reason)
}

func (bc *BaseContract) TokenBalanceSub(address *types.Address, amount *big.Int, reason string) error {
	return ledger.TokenBalanceSub(bc.stub, bc.config.GetSymbol(), address, amount, reason)
}

func (bc *BaseContract) TokenBalanceSubWithTicker(address *types.Address, amount *big.Int, ticker string, reason string) error {
	return ledger.TokenBalanceSubWithTicker(bc.stub, bc.config.GetSymbol(), address, amount, ticker, reason)
}

func (bc *BaseContract) TokenBalanceGetLocked(address *types.Address) (*big.Int, error) {
	return ledger.TokenBalanceGetLocked(bc.stub, address)
}

func (bc *BaseContract) TokenBalanceLock(address *types.Address, amount *big.Int) error {
	return ledger.TokenBalanceLock(bc.stub, bc.config.GetSymbol(), address, amount)
}

func (bc *BaseContract) TokenBalanceUnlock(address *types.Address, amount *big.Int) error {
	return ledger.TokenBalanceUnlock(bc.stub, bc.config.GetSymbol(), address, amount)
}

func (bc *BaseContract) TokenBalanceTransferLocked(from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.TokenBalanceTransferLocked(bc.stub, bc.config.GetSymbol(), from, to, amount, reason)
}

func (bc *BaseContract) TokenBalanceBurnLocked(address *types.Address, amount *big.Int, reason string) error {
	return ledger.TokenBalanceBurnLocked(bc.stub, bc.config.GetSymbol(), address, amount, reason)
}

func (bc *BaseContract) AllowedBalanceGet(token string, address *types.Address) (*big.Int, error) {
	return ledger.AllowedBalanceGet(bc.stub, token, address)
}

func (bc *BaseContract) AllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.AllowedBalanceAdd(bc.stub, token, address, amount, reason)
}

func (bc *BaseContract) AllowedBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.AllowedBalanceSub(bc.stub, token, address, amount, reason)
}

func (bc *BaseContract) AllowedIndustrialBalanceTransfer(from *types.Address, to *types.Address, industrialAssets []*pb.Asset, reason string) error {
	return ledger.AllowedIndustrialBalanceTransfer(bc.stub, from, to, industrialAssets, reason)
}

func (bc *BaseContract) AllowedIndustrialBalanceAdd(address *types.Address, industrialAssets []*pb.Asset, reason string) error {
	return ledger.AllowedIndustrialBalanceAdd(bc.stub, address, industrialAssets, reason)
}

func (bc *BaseContract) AllowedIndustrialBalanceSub(address *types.Address, industrialAssets []*pb.Asset, reason string) error {
	return ledger.AllowedIndustrialBalanceSub(bc.stub, address, industrialAssets, reason)
}

func (bc *BaseContract) AllowedBalanceGetLocked(token string, address *types.Address) (*big.Int, error) {
	return ledger.AllowedBalanceGetLocked(bc.stub, token, address)
}

func (bc *BaseContract) AllowedBalanceLock(token string, address *types.Address, amount *big.Int) error {
	return ledger.AllowedBalanceLock(bc.stub, bc.config.GetSymbol(), token, address, amount)
}

func (bc *BaseContract) AllowedBalanceUnLock(token string, address *types.Address, amount *big.Int) error {
	return ledger.AllowedBalanceUnlock(bc.stub, bc.config.GetSymbol(), token, address, amount)
}

func (bc *BaseContract) AllowedBalanceTransferLocked(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.AllowedBalanceTransferLocked(bc.stub, token, from, to, amount, reason)
}

func (bc *BaseContract) AllowedBalanceBurnLocked(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.AllowedBalanceBurnLocked(bc.stub, token, address, amount, reason)
}

func (bc *BaseContract) IndustrialBalanceGetLocked(address *types.Address) (map[string]string, error) {
	return ledger.IndustrialBalanceGetLocked(bc.stub, address)
}

func (bc *BaseContract) IndustrialBalanceLock(token string, address *types.Address, amount *big.Int) error {
	return ledger.IndustrialBalanceLock(bc.stub, bc.config.GetSymbol(), token, address, amount)
}

func (bc *BaseContract) IndustrialBalanceUnLock(token string, address *types.Address, amount *big.Int) error {
	return ledger.IndustrialBalanceUnlock(bc.stub, bc.config.GetSymbol(), token, address, amount)
}

func (bc *BaseContract) IndustrialBalanceTransferLocked(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return ledger.IndustrialBalanceTransferLocked(bc.stub, bc.config.GetSymbol(), token, from, to, amount, reason)
}

func (bc *BaseContract) IndustrialBalanceBurnLocked(token string, address *types.Address, amount *big.Int, reason string) error {
	return ledger.IndustrialBalanceBurnLocked(bc.stub, bc.config.GetSymbol(), token, address, amount, reason)
}

func (bc *BaseContract) AllowedBalanceGetAll(address *types.Address) (map[string]string, error) {
	return ledger.AllowedBalanceGetAll(bc.stub, address)
}
