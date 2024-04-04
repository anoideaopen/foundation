package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	mathbig "math/big"
	"runtime/debug"
	"strings"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"golang.org/x/crypto/sha3"
)

const (
	// ErrIncorrectSwap is a reason for multiswap
	ErrIncorrectSwap = "incorrect swap"
	// ErrIncorrectKey is a reason for multiswap
	ErrIncorrectKey = "incorrect key"

	userSideTimeout  = 10800 // 3 hours
	robotSideTimeout = 300   // 5 minutes
)

func swapAnswer(stub *batchStub, swap *proto.Swap) (r *proto.SwapResponse) {
	r = &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: "panic swapAnswer"}}
	defer func() {
		if rc := recover(); rc != nil {
			log.Println("panic swapAnswer: " + hex.EncodeToString(swap.Id) + "\n" + string(debug.Stack()))
		}
	}()

	ts, err := stub.GetTxTimestamp()
	if err != nil {
		return &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: err.Error()}}
	}
	txStub := stub.newTxStub(hex.EncodeToString(swap.Id))

	swap.Creator = []byte("0000")
	swap.Timeout = ts.Seconds + robotSideTimeout

	switch {
	case swap.TokenSymbol() == swap.From:
		// nothing to do
	case swap.TokenSymbol() == swap.To:
		if err = balance.Sub(txStub, balance.BalanceTypeGiven, swap.From, "", new(mathbig.Int).SetBytes(swap.Amount)); err != nil {
			return &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: err.Error()}}
		}
	default:
		return &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: ErrIncorrectSwap}}
	}

	if err = SwapSave(txStub, hex.EncodeToString(swap.Id), swap); err != nil {
		return &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swap.Id, Writes: writes}
}

func swapRobotDone(stub *batchStub, swapID []byte, key string) (r *proto.SwapResponse) {
	r = &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: "panic swapRobotDone"}}
	defer func() {
		if rc := recover(); rc != nil {
			log.Println("panic swapRobotDone: " + hex.EncodeToString(swapID) + "\n" + string(debug.Stack()))
		}
	}()

	txStub := stub.newTxStub(hex.EncodeToString(swapID))
	s, err := SwapLoad(txStub, hex.EncodeToString(swapID))
	if err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}
	hash := sha3.Sum256([]byte(key))
	if !bytes.Equal(s.Hash, hash[:]) {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: ErrIncorrectKey}}
	}

	if s.TokenSymbol() == s.From {
		if err = balance.Add(txStub, balance.BalanceTypeGiven, s.To, "", new(mathbig.Int).SetBytes(s.Amount)); err != nil {
			return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
		}
	}
	if err = SwapDel(txStub, hex.EncodeToString(swapID)); err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swapID, Writes: writes}
}

func swapUserDone(bci BaseContractInterface, swapID string, key string) peer.Response {
	s, err := SwapLoad(bci.GetStub(), swapID)
	if err != nil {
		return shim.Error(err.Error())
	}
	hash := sha3.Sum256([]byte(key))
	if !bytes.Equal(s.Hash, hash[:]) {
		return shim.Error(ErrIncorrectKey)
	}

	if bytes.Equal(s.Creator, s.Owner) {
		return shim.Error(ErrIncorrectSwap)
	}
	if s.TokenSymbol() == s.From {
		if err = bci.AllowedBalanceAdd(s.Token, types.AddrFromBytes(s.Owner), new(big.Int).SetBytes(s.Amount), "swap done"); err != nil {
			return shim.Error(err.Error())
		}
	} else {
		if err = bci.tokenBalanceAdd(types.AddrFromBytes(s.Owner), new(big.Int).SetBytes(s.Amount), s.Token); err != nil {
			return shim.Error(err.Error())
		}
	}

	if err = SwapDel(bci.GetStub(), swapID); err != nil {
		return shim.Error(err.Error())
	}
	e := strings.Join([]string{s.From, swapID, key}, "\t")
	if err = bci.GetStub().SetEvent("key", []byte(e)); err != nil {
		return shim.Error(err.Error())
	}

	// This code implements a callback which notifies that Swap was made.
	// This callback handles direct (move tokens to other channel) or
	// reverse (move tokens from other channel back) Swaps.
	// If you want to catch that events you need implement
	// method `OnSwapDoneEvent` in chaincode.
	// This code is for chaincode PFT, for handling user bar tokens balance changes.
	if f, ok := bci.(interface {
		OnSwapDoneEvent(
			token string,
			owner *types.Address,
			amount *big.Int,
		)
	}); ok {
		f.OnSwapDoneEvent(
			s.Token,
			types.AddrFromBytes(s.Owner),
			new(big.Int).SetBytes(s.Amount),
		)
	}

	return shim.Success(nil)
}

// QuerySwapGet returns swap by id
func (bc *BaseContract) QuerySwapGet(swapID string) (*proto.Swap, error) {
	swap, err := SwapLoad(bc.GetStub(), swapID)
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
		return "", errors.New(ErrIncorrectSwap)
	}

	if err = SwapSave(bc.GetStub(), bc.GetStub().GetTxID(), &s); err != nil {
		return "", err
	}

	if btchTxStub, ok := bc.stub.(*BatchTxStub); ok {
		btchTxStub.swaps = append(btchTxStub.swaps, &s)
	}
	return bc.GetStub().GetTxID(), nil
}

// TxSwapCancel cancels swap
func (bc *BaseContract) TxSwapCancel(_ *types.Sender, swapID string) error { // sender
	s, err := SwapLoad(bc.GetStub(), swapID)
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
	return SwapDel(bc.GetStub(), swapID)
}

// SwapLoad returns swap by id
func SwapLoad(stub shim.ChaincodeStubInterface, swapID string) (*proto.Swap, error) {
	key, err := stub.CreateCompositeKey("swaps", []string{swapID})
	if err != nil {
		return nil, err
	}
	data, err := stub.GetState(key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("swap doesn't exist by key %s", swapID)
	}
	var s proto.Swap
	if err = pb.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// SwapSave saves swap
func SwapSave(stub shim.ChaincodeStubInterface, swapID string, s *proto.Swap) error {
	key, err := stub.CreateCompositeKey("swaps", []string{swapID})
	if err != nil {
		return err
	}
	data, err := pb.Marshal(s)
	if err != nil {
		return err
	}
	return stub.PutState(key, data)
}

// SwapDel deletes swap
func SwapDel(stub shim.ChaincodeStubInterface, swapID string) error {
	key, err := stub.CreateCompositeKey("swaps", []string{swapID})
	if err != nil {
		return err
	}
	return stub.DelState(key)
}
