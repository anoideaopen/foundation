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
	txStub := stub.NewTxCacheStub(hex.EncodeToString(swap.Id))

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

	if err = Save(txStub, hex.EncodeToString(swap.Id), swap); err != nil {
		return &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swap.Id, Writes: writes}
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
	if !bytes.Equal(s.Hash, hash[:]) {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: ErrIncorrectKey}}
	}

	if s.TokenSymbol() == s.From {
		if err = balance.Add(txStub, balance.BalanceTypeGiven, s.To, "", new(mathbig.Int).SetBytes(s.Amount)); err != nil {
			return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
		}
	}
	if err = Delete(txStub, hex.EncodeToString(swapID)); err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swapID, Writes: writes}
}

func UserDone(bci BaseContractInterface, swapID string, key string) peer.Response {
	s, err := Load(bci.GetStub(), swapID)
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
		if err = bci.TokenBalanceAdd(types.AddrFromBytes(s.Owner), new(big.Int).SetBytes(s.Amount), s.Token); err != nil {
			return shim.Error(err.Error())
		}
	}

	if err = Delete(bci.GetStub(), swapID); err != nil {
		return shim.Error(err.Error())
	}
	e := strings.Join([]string{s.From, swapID, key}, "\t")
	if err = bci.GetStub().SetEvent(SwapKeyEvent, []byte(e)); err != nil {
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
