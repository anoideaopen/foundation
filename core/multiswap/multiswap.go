package multiswap

import (
	"bytes"
	"crypto/sha3"
	"encoding/hex"
	"errors"
	"log"
	mathbig "math/big"
	"runtime/debug"
	"strings"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/ledger"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	pb "google.golang.org/protobuf/proto"
)

const (
	// MultiSwapCompositeType is a composite key for multiswap
	MultiSwapCompositeType = "multi_swap"
	// MultiSwapKeyEvent is a reason for multiswap
	MultiSwapKeyEvent = "multi_swap_key"

	// ErrIncorrectMultiSwap is a reason for multiswap
	ErrIncorrectMultiSwap = "incorrect swap"
	// ErrIncorrectMultiSwapKey is a reason for multiswap
	ErrIncorrectMultiSwapKey = "incorrect key"
)

// BaseContractInterface represents BaseContract interface
type BaseContractInterface interface {
	GetStub() shim.ChaincodeStubInterface
	TokenBalanceAddWithTicker(address *types.Address, amount *big.Int, ticker string, reason string) error
	AllowedIndustrialBalanceAdd(address *types.Address, industrialAssets []*proto.Asset, reason string) error
}

func Answer(stub *cachestub.BatchCacheStub, swap *proto.MultiSwap, robotSideTimeout int64) (r *proto.SwapResponse) {
	r = &proto.SwapResponse{Id: swap.GetId(), Error: &proto.ResponseError{Error: "panic multiSwapAnswer"}}
	defer func() {
		if rc := recover(); rc != nil {
			log.Println("panic multiSwapAnswer: " + hex.EncodeToString(swap.GetId()) + "\n" + string(debug.Stack()))
		}
	}()

	ts, err := stub.GetTxTimestamp()
	if err != nil {
		return &proto.SwapResponse{Id: swap.GetId(), Error: &proto.ResponseError{Error: err.Error()}}
	}
	txStub := stub.NewTxCacheStub(hex.EncodeToString(swap.GetId()), ts)

	swap.Creator = []byte("0000")
	swap.Timeout = ts.GetSeconds() + robotSideTimeout

	switch {
	case swap.GetToken() == swap.GetFrom():
		// nothing to do
	case swap.GetToken() == swap.GetTo():
		for _, asset := range swap.GetAssets() {
			if err = balance.Sub(txStub, balance.BalanceTypeGiven, strings.ToUpper(swap.GetFrom()), "", new(mathbig.Int).SetBytes(asset.GetAmount())); err != nil {
				return &proto.SwapResponse{Id: swap.GetId(), Error: &proto.ResponseError{Error: err.Error()}}
			}
		}
	default:
		return &proto.SwapResponse{Id: swap.GetId(), Error: &proto.ResponseError{Error: ErrIncorrectMultiSwap}}
	}

	if err = Save(txStub, hex.EncodeToString(swap.GetId()), swap); err != nil {
		return &proto.SwapResponse{Id: swap.GetId(), Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swap.GetId(), Writes: writes}
}

func RobotDone(stub *cachestub.BatchCacheStub, swapID []byte, key string) (r *proto.SwapResponse) {
	r = &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: "panic multiSwapRobotDone"}}
	defer func() {
		if rc := recover(); rc != nil {
			log.Println("panic multiSwapRobotDone: " + hex.EncodeToString(swapID) + "\n" + string(debug.Stack()))
		}
	}()

	ts, err := stub.GetTxTimestamp()
	if err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}

	txStub := stub.NewTxCacheStub(hex.EncodeToString(swapID), ts)
	swap, err := Load(txStub, hex.EncodeToString(swapID))
	if err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}
	hash := sha3.Sum256([]byte(key))
	if !bytes.Equal(swap.GetHash(), hash[:]) {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: ErrIncorrectMultiSwapKey}}
	}

	if swap.GetToken() == swap.GetFrom() {
		for _, asset := range swap.GetAssets() {
			if err = balance.Add(txStub, balance.BalanceTypeGiven, strings.ToUpper(swap.GetTo()), "", new(mathbig.Int).SetBytes(asset.GetAmount())); err != nil {
				return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
			}
		}
	}

	if err = Delete(txStub, hex.EncodeToString(swapID)); err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swapID, Writes: writes}
}

type OnMultiSwapDoneEventListener interface {
	OnMultiSwapDoneEvent(
		token string,
		owner *types.Address,
		assets []*proto.Asset,
	)
}

func UserDone(bci any, stub shim.ChaincodeStubInterface, symbol string, swapID string, key string) *peer.Response {
	swap, err := Load(stub, swapID)
	if err != nil {
		return shim.Error(err.Error())
	}
	hash := sha3.Sum256([]byte(key))
	if !bytes.Equal(swap.GetHash(), hash[:]) {
		return shim.Error(ErrIncorrectMultiSwapKey)
	}

	if bytes.Equal(swap.GetCreator(), swap.GetOwner()) {
		return shim.Error(ErrIncorrectMultiSwap)
	}
	if swap.GetToken() == swap.GetFrom() {
		if err = ledger.AllowedIndustrialBalanceAdd(stub, types.AddrFromBytes(swap.GetOwner()), swap.GetAssets(), "multi-swap done"); err != nil {
			return shim.Error(err.Error())
		}
	} else {
		for _, asset := range swap.GetAssets() {
			if err = ledger.TokenBalanceAddWithTicker(stub, symbol, types.AddrFromBytes(swap.GetOwner()), new(big.Int).SetBytes(asset.GetAmount()), asset.GetGroup(), "reverse multi-swap done"); err != nil {
				return shim.Error(err.Error())
			}
		}
	}

	if err = Delete(stub, swapID); err != nil {
		return shim.Error(err.Error())
	}
	e := strings.Join([]string{swap.GetFrom(), swapID, key}, "\t")
	if err = stub.SetEvent(MultiSwapKeyEvent, []byte(e)); err != nil {
		return shim.Error(err.Error())
	}

	// This code implements a callback which notifies that MultiSwap was made.
	// This callback handles direct (move tokens to other channel) or
	// reverse (move tokens from other channel back) MultiSwaps.
	// If you want to catch that events you need implement
	// method `OnMultiSwapDoneEvent` in chaincode.
	// This code is for chaincode PFT, for handling user bar tokens balance changes.
	if listener, ok := bci.(OnMultiSwapDoneEventListener); ok {
		listener.OnMultiSwapDoneEvent(
			swap.GetToken(),
			types.AddrFromBytes(swap.GetOwner()),
			swap.GetAssets(),
		)
	}

	return shim.Success(nil)
}

// Load loads multiswap from the ledger
func Load(stub shim.ChaincodeStubInterface, swapID string) (*proto.MultiSwap, error) {
	key, err := stub.CreateCompositeKey(MultiSwapCompositeType, []string{swapID})
	if err != nil {
		return nil, err
	}
	data, err := stub.GetState(key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, errors.New("multiswap doesn't exist")
	}
	var swap proto.MultiSwap
	if err = pb.Unmarshal(data, &swap); err != nil {
		return nil, err
	}
	return &swap, nil
}

// Save saves multiswap to the ledger
func Save(stub shim.ChaincodeStubInterface, swapID string, swap *proto.MultiSwap) error {
	key, err := stub.CreateCompositeKey(MultiSwapCompositeType, []string{swapID})
	if err != nil {
		return err
	}
	data, err := pb.Marshal(swap)
	if err != nil {
		return err
	}
	return stub.PutState(key, data)
}

// Delete deletes multiswap from the ledger
func Delete(stub shim.ChaincodeStubInterface, swapID string) error {
	key, err := stub.CreateCompositeKey(MultiSwapCompositeType, []string{swapID})
	if err != nil {
		return err
	}
	return stub.DelState(key)
}
