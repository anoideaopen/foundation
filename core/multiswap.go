package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	mathbig "math/big"
	"strings"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/multiswap"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

// multiSwapDoneHandler processes a request to mark multiple swaps as done.
// If the ChainCode is configured to disable multi swaps, it will immediately return an error.
//
// It loads initial arguments and then proceeds to execute the multi-swap user done logic.
//
// Returns a shim.Success response if the multi-swap done logic executes successfully.
// Otherwise, it returns a shim.Error response.
func (cc *Chaincode) multiSwapDoneHandler(
	args []string,
) peer.Response {
	if cc.contract.ContractConfig().GetOptions().GetDisableMultiSwaps() {
		return shim.Error("handling multi-swap done failed, " + ErrMultiSwapDisabled.Error())
	}

	return multiswap.UserDone(cc.contract, args[0], args[1])
}

// QueryMultiSwapGet - returns multiswap by id
func (bc *BaseContract) QueryMultiSwapGet(swapID string) (*proto.MultiSwap, error) {
	swap, err := multiswap.Load(bc.GetStub(), swapID)
	if err != nil {
		return nil, err
	}
	return swap, nil
}

// TxMultiSwapBegin - creates multiswap
func (bc *BaseContract) TxMultiSwapBegin(sender *types.Sender, token string, multiSwapAssets types.MultiSwapAssets, contractTo string, hash types.Hex) (string, error) {
	id, err := hex.DecodeString(bc.GetStub().GetTxID())
	if err != nil {
		return "", err
	}
	ts, err := bc.GetStub().GetTxTimestamp()
	if err != nil {
		return "", err
	}
	assets, err := types.ConvertToAsset(multiSwapAssets.Assets)
	if err != nil {
		return "", err
	}
	if len(assets) == 0 {
		return "", errors.New("assets can't be empty")
	}

	swap := proto.MultiSwap{
		Id:      id,
		Creator: sender.Address().Bytes(),
		Owner:   sender.Address().Bytes(),
		Assets:  assets,
		Token:   token,
		From:    bc.config.GetSymbol(),
		To:      contractTo,
		Hash:    hash,
		Timeout: ts.GetSeconds() + userSideTimeout,
	}

	switch {
	case swap.GetToken() == swap.GetFrom():
		for _, asset := range swap.GetAssets() {
			if err = bc.TokenBalanceSubWithTicker(types.AddrFromBytes(swap.GetOwner()), new(big.Int).SetBytes(asset.GetAmount()), asset.GetGroup(), "multi-swap begin"); err != nil {
				return "", err
			}
		}
	case swap.GetToken() == swap.GetTo():
		if err = bc.AllowedIndustrialBalanceSub(types.AddrFromBytes(swap.GetOwner()), swap.GetAssets(), "reverse multi-swap begin"); err != nil {
			return "", err
		}
	default:
		return "", errors.New(multiswap.ErrIncorrectMultiSwap)
	}

	if err = multiswap.Save(bc.GetStub(), bc.GetStub().GetTxID(), &swap); err != nil {
		return "", err
	}

	if btchTxStub, ok := bc.stub.(*cachestub.TxCacheStub); ok {
		btchTxStub.MultiSwaps = append(btchTxStub.MultiSwaps, &swap)
	}
	return bc.GetStub().GetTxID(), nil
}

// TxMultiSwapCancel - cancels multiswap
func (bc *BaseContract) TxMultiSwapCancel(sender *types.Sender, swapID string) error {
	swap, err := multiswap.Load(bc.GetStub(), swapID)
	if err != nil {
		return err
	}
	if !bytes.Equal(swap.GetCreator(), sender.Address().Bytes()) {
		return fmt.Errorf("unauthorized, multiswap creator %s not eq sender %s",
			string(swap.GetCreator()), sender.Address().String())
	}

	ts, err := bc.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	if swap.GetTimeout() > ts.GetSeconds() {
		return errors.New("wait for timeout to end")
	}

	switch {
	case bytes.Equal(swap.GetCreator(), swap.GetOwner()) && swap.GetToken() == swap.GetFrom():
		for _, asset := range swap.GetAssets() {
			if err = bc.TokenBalanceAddWithTicker(types.AddrFromBytes(swap.GetOwner()), new(big.Int).SetBytes(asset.GetAmount()), asset.GetGroup(), "multi-swap cancel"); err != nil {
				return err
			}
		}
	case bytes.Equal(swap.GetCreator(), swap.GetOwner()) && swap.GetToken() == swap.GetTo():
		if err = bc.AllowedIndustrialBalanceAdd(types.AddrFromBytes(swap.GetOwner()), swap.GetAssets(), "reverse multi-swap cancel"); err != nil {
			return err
		}
	case bytes.Equal(swap.GetCreator(), []byte("0000")) && swap.GetToken() == swap.GetTo():
		for _, asset := range swap.GetAssets() {
			if err = balance.Add(bc.GetStub(), balance.BalanceTypeGiven, strings.ToUpper(swap.GetFrom()), "", new(mathbig.Int).SetBytes(asset.GetAmount())); err != nil {
				return err
			}
		}
	}

	return multiswap.Delete(bc.GetStub(), swapID)
}

// QueryGroupBalanceOf - returns balance of the token for user address
func (bc *BaseContract) QueryGroupBalanceOf(address *types.Address) (map[string]string, error) {
	return bc.IndustrialBalanceGet(address)
}
