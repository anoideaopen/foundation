package swap

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	mathbig "math/big"
	"runtime/debug"
	"strings"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/ledger"
	"github.com/anoideaopen/foundation/core/routing/reflectx"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"golang.org/x/crypto/sha3"
)

const (
	// SwapCompositeType is a composite key for swap
	SwapCompositeType = "swaps"
	// SwapKeyEvent is a reason for swap
	SwapKeyEvent = "key"

	// ErrIncorrectSwap is a reason for multiswap
	ErrIncorrectSwap = "incorrect swap"
	// ErrIncorrectKey is a reason for multiswap
	ErrIncorrectKey = "incorrect key"
)

// BaseContractInterface represents BaseContract interface
type BaseContractInterface interface {
	GetStub() shim.ChaincodeStubInterface
	AllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error
	TokenBalanceAdd(address *types.Address, amount *big.Int, reason string) error
}

func Answer(stub *cachestub.BatchCacheStub, swap *proto.Swap, robotSideTimeout int64) (r *proto.SwapResponse) {
	r = &proto.SwapResponse{Id: swap.GetId(), Error: &proto.ResponseError{Error: "panic swapAnswer"}}
	defer func() {
		if rc := recover(); rc != nil {
			log.Println("panic swapAnswer: " + hex.EncodeToString(swap.GetId()) + "\n" + string(debug.Stack()))
		}
	}()

	ts, err := stub.GetTxTimestamp()
	if err != nil {
		return &proto.SwapResponse{Id: swap.GetId(), Error: &proto.ResponseError{Error: err.Error()}}
	}
	txStub := stub.NewTxCacheStub(hex.EncodeToString(swap.GetId()))

	swap.Creator = []byte("0000")
	swap.Timeout = ts.GetSeconds() + robotSideTimeout

	switch {
	case swap.TokenSymbol() == swap.GetFrom():
		// nothing to do
	case swap.TokenSymbol() == swap.GetTo():
		if err = balance.Sub(txStub, balance.BalanceTypeGiven, strings.ToUpper(swap.GetFrom()), "", new(mathbig.Int).SetBytes(swap.GetAmount())); err != nil {
			return &proto.SwapResponse{Id: swap.GetId(), Error: &proto.ResponseError{Error: err.Error()}}
		}
	default:
		return &proto.SwapResponse{Id: swap.GetId(), Error: &proto.ResponseError{Error: ErrIncorrectSwap}}
	}

	if err = Save(txStub, hex.EncodeToString(swap.GetId()), swap); err != nil {
		return &proto.SwapResponse{Id: swap.GetId(), Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swap.GetId(), Writes: writes}
}

func RobotDone(stub *cachestub.BatchCacheStub, swapID []byte, key string) (r *proto.SwapResponse) {
	r = &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: "panic swapRobotDone"}}
	defer func() {
		if rc := recover(); rc != nil {
			log.Println("panic swapRobotDone: " + hex.EncodeToString(swapID) + "\n" + string(debug.Stack()))
		}
	}()

	txStub := stub.NewTxCacheStub(hex.EncodeToString(swapID))
	s, err := Load(txStub, hex.EncodeToString(swapID))
	if err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}
	hash := sha3.Sum256([]byte(key))
	if !bytes.Equal(s.GetHash(), hash[:]) {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: ErrIncorrectKey}}
	}

	if s.TokenSymbol() == s.GetFrom() {
		if err = balance.Add(txStub, balance.BalanceTypeGiven, strings.ToUpper(s.GetTo()), "", new(mathbig.Int).SetBytes(s.GetAmount())); err != nil {
			return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
		}
	}
	if err = Delete(txStub, hex.EncodeToString(swapID)); err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swapID, Writes: writes}
}

type OnSwapDoneEventListener interface {
	OnSwapDoneEvent(
		token string,
		owner *types.Address,
		amount *big.Int,
	)
}

func UserDone(bci any, stub shim.ChaincodeStubInterface, swapID string, key string) peer.Response {
	s, err := Load(stub, swapID)
	if err != nil {
		return shim.Error(err.Error())
	}
	hash := sha3.Sum256([]byte(key))
	if !bytes.Equal(s.GetHash(), hash[:]) {
		return shim.Error(ErrIncorrectKey)
	}

	if bytes.Equal(s.GetCreator(), s.GetOwner()) {
		return shim.Error(ErrIncorrectSwap)
	}
	if s.TokenSymbol() == s.GetFrom() {
		if err = ledger.AllowedBalanceAdd(stub, s.GetToken(), types.AddrFromBytes(s.GetOwner()), new(big.Int).SetBytes(s.GetAmount()), "swap done"); err != nil {
			return shim.Error(err.Error())
		}
	} else {
		if err = ledger.TokenBalanceAdd(stub, types.AddrFromBytes(s.GetOwner()), new(big.Int).SetBytes(s.GetAmount()), s.GetToken()); err != nil {
			return shim.Error(err.Error())
		}
	}

	if err = Delete(stub, swapID); err != nil {
		return shim.Error(err.Error())
	}
	e := strings.Join([]string{s.GetFrom(), swapID, key}, "\t")
	if err = stub.SetEvent(SwapKeyEvent, []byte(e)); err != nil {
		return shim.Error(err.Error())
	}

	// This code implements a callback which notifies that Swap was made.
	// This callback handles direct (move tokens to other channel) or
	// reverse (move tokens from other channel back) Swaps.
	// If you want to catch that events you need implement
	// method `OnSwapDoneEvent` in chaincode.
	// This code is for chaincode PFT, for handling user bar tokens balance changes.
	if _, ok := bci.(OnSwapDoneEventListener); ok {
		bciClone, ok := reflectx.Clone(bci).(OnSwapDoneEventListener)
		if !ok {
			return shim.Error("failed to clone bci")
		}

		if stubSetter, ok := bciClone.(reflectx.StubSetter); ok {
			stubSetter.SetStub(stub)
		}

		bciClone.OnSwapDoneEvent(
			s.GetToken(),
			types.AddrFromBytes(s.GetOwner()),
			new(big.Int).SetBytes(s.GetAmount()),
		)
	}

	return shim.Success(nil)
}

// Load returns swap by id
func Load(stub shim.ChaincodeStubInterface, swapID string) (*proto.Swap, error) {
	key, err := stub.CreateCompositeKey(SwapCompositeType, []string{swapID})
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

// Save saves swap
func Save(stub shim.ChaincodeStubInterface, swapID string, s *proto.Swap) error {
	key, err := stub.CreateCompositeKey(SwapCompositeType, []string{swapID})
	if err != nil {
		return err
	}
	data, err := pb.Marshal(s)
	if err != nil {
		return err
	}
	return stub.PutState(key, data)
}

// Delete deletes swap
func Delete(stub shim.ChaincodeStubInterface, swapID string) error {
	key, err := stub.CreateCompositeKey(SwapCompositeType, []string{swapID})
	if err != nil {
		return err
	}
	return stub.DelState(key)
}
