package multiswap

import (
	"bytes"
	"encoding/hex"
	"errors"
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
	r = &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: "panic multiSwapAnswer"}}
	defer func() {
		if rc := recover(); rc != nil {
			log.Println("panic multiSwapAnswer: " + hex.EncodeToString(swap.Id) + "\n" + string(debug.Stack()))
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
	case swap.Token == swap.From:
		// nothing to do
	case swap.Token == swap.To:
		for _, asset := range swap.Assets {
			if err = balance.Sub(txStub, balance.BalanceTypeGiven, swap.From, "", new(mathbig.Int).SetBytes(asset.Amount)); err != nil {
				return &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: err.Error()}}
			}
		}
	default:
		return &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: ErrIncorrectMultiSwap}}
	}

	if err = Save(txStub, hex.EncodeToString(swap.Id), swap); err != nil {
		return &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swap.Id, Writes: writes}
}

func RobotDone(stub *cachestub.BatchCacheStub, swapID []byte, key string) (r *proto.SwapResponse) {
	r = &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: "panic multiSwapRobotDone"}}
	defer func() {
		if rc := recover(); rc != nil {
			log.Println("panic multiSwapRobotDone: " + hex.EncodeToString(swapID) + "\n" + string(debug.Stack()))
		}
	}()

	txStub := stub.NewTxCacheStub(hex.EncodeToString(swapID))
	swap, err := Load(txStub, hex.EncodeToString(swapID))
	if err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}
	hash := sha3.Sum256([]byte(key))
	if !bytes.Equal(swap.Hash, hash[:]) {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: ErrIncorrectMultiSwapKey}}
	}

	if swap.Token == swap.From {
		for _, asset := range swap.Assets {
			if err = balance.Add(txStub, balance.BalanceTypeGiven, swap.To, "", new(mathbig.Int).SetBytes(asset.Amount)); err != nil {
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

func UserDone(bc BaseContractInterface, swapID string, key string) peer.Response {
	swap, err := Load(bc.GetStub(), swapID)
	if err != nil {
		return shim.Error(err.Error())
	}
	hash := sha3.Sum256([]byte(key))
	if !bytes.Equal(swap.Hash, hash[:]) {
		return shim.Error(ErrIncorrectMultiSwapKey)
	}

	if bytes.Equal(swap.Creator, swap.Owner) {
		return shim.Error(ErrIncorrectMultiSwap)
	}
	if swap.Token == swap.From {
		if err = bc.AllowedIndustrialBalanceAdd(types.AddrFromBytes(swap.Owner), swap.Assets, "multi-swap done"); err != nil {
			return shim.Error(err.Error())
		}
	} else {
		for _, asset := range swap.Assets {
			if err = bc.TokenBalanceAddWithTicker(types.AddrFromBytes(swap.Owner), new(big.Int).SetBytes(asset.Amount), asset.Group, "reverse multi-swap done"); err != nil {
				return shim.Error(err.Error())
			}
		}
	}

	if err = Delete(bc.GetStub(), swapID); err != nil {
		return shim.Error(err.Error())
	}
	e := strings.Join([]string{swap.From, swapID, key}, "\t")
	if err = bc.GetStub().SetEvent(MultiSwapKeyEvent, []byte(e)); err != nil {
		return shim.Error(err.Error())
	}

	// This code implements a callback which notifies that MultiSwap was made.
	// This callback handles direct (move tokens to other channel) or
	// reverse (move tokens from other channel back) MultiSwaps.
	// If you want to catch that events you need implement
	// method `OnMultiSwapDoneEvent` in chaincode.
	// This code is for chaincode PFT, for handling user bar tokens balance changes.
	if f, ok := bc.(interface {
		OnMultiSwapDoneEvent(
			token string,
			owner *types.Address,
			assets []*proto.Asset,
		)
	}); ok {
		f.OnMultiSwapDoneEvent(
			swap.Token,
			types.AddrFromBytes(swap.Owner),
			swap.Assets,
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
