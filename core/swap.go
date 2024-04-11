package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	mathbig "math/big"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/swap"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
)

const (
	userSideTimeout = 10800 // 3 hours
)

// QuerySwapGet returns swap by id
func (bc *BaseContract) QuerySwapGet(swapID string) (*proto.Swap, error) {
	swap, err := swap.SwapLoad(bc.GetStub(), swapID)
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
		From:    bc.config.Symbol,
		To:      contractTo,
		Hash:    hash,
		Timeout: ts.Seconds + userSideTimeout,
	}

	switch {
	case s.TokenSymbol() == s.From:
		if err = bc.TokenBalanceSubWithTicker(types.AddrFromBytes(s.Owner), amount, s.Token, "swap begin"); err != nil {
			return "", err
		}
	case s.TokenSymbol() == s.To:
		if err = bc.AllowedBalanceSub(s.Token, types.AddrFromBytes(s.Owner), amount, "reverse swap begin"); err != nil {
			return "", err
		}
	default:
		return "", errors.New(swap.ErrIncorrectSwap)
	}

	if err = swap.SwapSave(bc.GetStub(), bc.GetStub().GetTxID(), &s); err != nil {
		return "", err
	}

	if btchTxStub, ok := bc.stub.(*cachestub.TxCacheStub); ok {
		btchTxStub.Swaps = append(btchTxStub.Swaps, &s)
	}
	return bc.GetStub().GetTxID(), nil
}

// TxSwapCancel cancels swap
func (bc *BaseContract) TxSwapCancel(_ *types.Sender, swapID string) error { // sender
	s, err := swap.SwapLoad(bc.GetStub(), swapID)
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
	case bytes.Equal(s.Creator, s.Owner) && s.TokenSymbol() == s.From:
		if err = bc.TokenBalanceAddWithTicker(types.AddrFromBytes(s.Owner), new(big.Int).SetBytes(s.Amount), s.Token, "swap cancel"); err != nil {
			return err
		}
	case bytes.Equal(s.Creator, s.Owner) && s.TokenSymbol() == s.To:
		if err = bc.AllowedBalanceAdd(s.Token, types.AddrFromBytes(s.Owner), new(big.Int).SetBytes(s.Amount), "reverse swap cancel"); err != nil {
			return err
		}
	case bytes.Equal(s.Creator, []byte("0000")) && s.TokenSymbol() == s.To:
		if err = balance.Add(bc.GetStub(), balance.BalanceTypeGiven, s.From, "", new(mathbig.Int).SetBytes(s.Amount)); err != nil {
			return err
		}
	}
	return swap.SwapDel(bc.GetStub(), swapID)
}
