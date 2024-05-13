package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	mathbig "math/big"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/swap"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

const (
	userSideTimeout = 10800 // 3 hours
)

// swapDoneHandler processes a request to mark a swap as done.
// If the ChainCode is configured to disable swaps, it will immediately return an error.
//
// It loads initial arguments and then proceeds to execute the swap user done logic.
//
// Returns a shim.Success response if the swap done logic executes successfully.
// Otherwise, it returns a shim.Error response.
func (cc *ChainCode) swapDoneHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	args []string,
	cfgBytes []byte,
) peer.Response {
	if cc.contract.ContractConfig().GetOptions().GetDisableSwaps() {
		return shim.Error("handling swap done failed, " + ErrSwapDisabled.Error())
	}

	contract, ok := copyContractWithConfig(traceCtx, cc.contract, stub, cfgBytes).(BaseContractInterface)
	if !ok {
		return shim.Error("unsupported contract type")
	}

	return swap.UserDone(contract, args[0], args[1])
}

// QuerySwapGet returns swap by id
func (bc *BaseContract) QuerySwapGet(swapID string) (*proto.Swap, error) {
	swap, err := swap.Load(bc.GetStub(), swapID)
	if err != nil {
		return nil, err
	}
	return swap, nil
}

// TxSwapBegin creates swap
func (bc *BaseContract) TxSwapBegin(
	sender *types.Sender,
	token string,
	contractTo string,
	amount *big.Int,
	hash types.Hex,
) (string, error) {
	id, err := hex.DecodeString(bc.GetStub().GetTxID())
	if err != nil {
		return "", err
	}
	ts, err := bc.GetStub().GetTxTimestamp()
	if err != nil {
		return "", err
	}
	s := proto.Swap{
		Id:      id,
		Creator: sender.Address().Bytes(),
		Owner:   sender.Address().Bytes(),
		Token:   token,
		Amount:  amount.Bytes(),
		From:    bc.config.GetSymbol(),
		To:      contractTo,
		Hash:    hash,
		Timeout: ts.GetSeconds() + userSideTimeout,
	}

	switch {
	case s.TokenSymbol() == s.GetFrom():
		if err = bc.TokenBalanceSubWithTicker(types.AddrFromBytes(s.GetOwner()), amount, s.GetToken(), "swap begin"); err != nil {
			return "", err
		}
	case s.TokenSymbol() == s.GetTo():
		if err = bc.AllowedBalanceSub(s.GetToken(), types.AddrFromBytes(s.GetOwner()), amount, "reverse swap begin"); err != nil {
			return "", err
		}
	default:
		return "", errors.New(swap.ErrIncorrectSwap)
	}

	if err = swap.Save(bc.GetStub(), bc.GetStub().GetTxID(), &s); err != nil {
		return "", err
	}

	if btchTxStub, ok := bc.stub.(*cachestub.TxCacheStub); ok {
		btchTxStub.Swaps = append(btchTxStub.Swaps, &s)
	}
	return bc.GetStub().GetTxID(), nil
}

// TxSwapCancel cancels swap
func (bc *BaseContract) TxSwapCancel(_ *types.Sender, swapID string) error { // sender
	s, err := swap.Load(bc.GetStub(), swapID)
	if err != nil {
		return err
	}

	// Very dangerous, bug in the cancel swap logic
	// PFI
	// code is commented out, swap and acl should be redesigned.
	// In the meantime, the site should ensure correctness of swapCancel calls
	// 1. filter out all swapCancel calls, except for those made on behalf of the site.
	// 2. Do not call swapCancel on the FROM channel until swapCancel has passed on the TO channel
	// if !bytes.Equal(s.Creator, sender.Address().Bytes()) {
	// return errors.New("unauthorized")
	// }
	// ts, err := bc.GetStub().GetTxTimestamp()
	// if err != nil {
	// return err
	// }
	// if s.Timeout > ts.Seconds {
	// return errors.New("wait for timeout to end")
	// }

	switch {
	case bytes.Equal(s.GetCreator(), s.GetOwner()) && s.TokenSymbol() == s.GetFrom():
		if err = bc.TokenBalanceAddWithTicker(types.AddrFromBytes(s.GetOwner()), new(big.Int).SetBytes(s.GetAmount()), s.GetToken(), "swap cancel"); err != nil {
			return err
		}
	case bytes.Equal(s.GetCreator(), s.GetOwner()) && s.TokenSymbol() == s.GetTo():
		if err = bc.AllowedBalanceAdd(s.GetToken(), types.AddrFromBytes(s.GetOwner()), new(big.Int).SetBytes(s.GetAmount()), "reverse swap cancel"); err != nil {
			return err
		}
	case bytes.Equal(s.GetCreator(), []byte("0000")) && s.TokenSymbol() == s.GetTo():
		if err = balance.Add(bc.GetStub(), balance.BalanceTypeGiven, s.GetFrom(), "", new(mathbig.Int).SetBytes(s.GetAmount())); err != nil {
			return err
		}
	}
	return swap.Delete(bc.GetStub(), swapID)
}
