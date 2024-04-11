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
)

func multiSwapAnswer(stub *cachestub.BatchCacheStub, swap *proto.MultiSwap) (r *proto.SwapResponse) {
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
		return &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: ErrIncorrectSwap}}
	}

	if err = MultiSwapSave(txStub, hex.EncodeToString(swap.Id), swap); err != nil {
		return &proto.SwapResponse{Id: swap.Id, Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swap.Id, Writes: writes}
}

func multiSwapRobotDone(stub *cachestub.BatchCacheStub, swapID []byte, key string) (r *proto.SwapResponse) {
	r = &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: "panic multiSwapRobotDone"}}
	defer func() {
		if rc := recover(); rc != nil {
			log.Println("panic multiSwapRobotDone: " + hex.EncodeToString(swapID) + "\n" + string(debug.Stack()))
		}
	}()

	txStub := stub.NewTxCacheStub(hex.EncodeToString(swapID))
	swap, err := MultiSwapLoad(txStub, hex.EncodeToString(swapID))
	if err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}
	hash := sha3.Sum256([]byte(key))
	if !bytes.Equal(swap.Hash, hash[:]) {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: ErrIncorrectKey}}
	}

	if swap.Token == swap.From {
		for _, asset := range swap.Assets {
			if err = balance.Add(txStub, balance.BalanceTypeGiven, swap.To, "", new(mathbig.Int).SetBytes(asset.Amount)); err != nil {
				return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
			}
		}
	}

	if err = MultiSwapDel(txStub, hex.EncodeToString(swapID)); err != nil {
		return &proto.SwapResponse{Id: swapID, Error: &proto.ResponseError{Error: err.Error()}}
	}
	writes, _ := txStub.Commit()
	return &proto.SwapResponse{Id: swapID, Writes: writes}
}

func multiSwapUserDone(bc BaseContractInterface, swapID string, key string) peer.Response {
	swap, err := MultiSwapLoad(bc.GetStub(), swapID)
	if err != nil {
		return shim.Error(err.Error())
	}
	hash := sha3.Sum256([]byte(key))
	if !bytes.Equal(swap.Hash, hash[:]) {
		return shim.Error(ErrIncorrectKey)
	}

	if bytes.Equal(swap.Creator, swap.Owner) {
		return shim.Error(ErrIncorrectSwap)
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

	if err = MultiSwapDel(bc.GetStub(), swapID); err != nil {
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

// QueryMultiSwapGet - returns multiswap by id
func (bc *BaseContract) QueryMultiSwapGet(swapID string) (*proto.MultiSwap, error) {
	swap, err := MultiSwapLoad(bc.GetStub(), swapID)
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
		From:    bc.config.Symbol,
		To:      contractTo,
		Hash:    hash,
		Timeout: ts.Seconds + userSideTimeout,
	}

	switch {
	case swap.Token == swap.From:
		for _, asset := range swap.Assets {
			if err = bc.TokenBalanceSubWithTicker(types.AddrFromBytes(swap.Owner), new(big.Int).SetBytes(asset.Amount), asset.Group, "multi-swap begin"); err != nil {
				return "", err
			}
		}
	case swap.Token == swap.To:
		if err = bc.AllowedIndustrialBalanceSub(types.AddrFromBytes(swap.Owner), swap.Assets, "reverse multi-swap begin"); err != nil {
			return "", err
		}
	default:
		return "", errors.New(ErrIncorrectSwap)
	}

	if err = MultiSwapSave(bc.GetStub(), bc.GetStub().GetTxID(), &swap); err != nil {
		return "", err
	}

	if btchTxStub, ok := bc.stub.(*cachestub.TxCacheStub); ok {
		btchTxStub.MultiSwaps = append(btchTxStub.MultiSwaps, &swap)
	}
	return bc.GetStub().GetTxID(), nil
}

// TxMultiSwapCancel - cancels multiswap
func (bc *BaseContract) TxMultiSwapCancel(sender *types.Sender, swapID string) error {
	swap, err := MultiSwapLoad(bc.GetStub(), swapID)
	if err != nil {
		return err
	}
	if !bytes.Equal(swap.Creator, sender.Address().Bytes()) {
		return fmt.Errorf("unauthorized, multiswap creator %s not eq sender %s",
			string(swap.Creator), sender.Address().String())
	}

	ts, err := bc.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	if swap.Timeout > ts.Seconds {
		return errors.New("wait for timeout to end")
	}

	switch {
	case bytes.Equal(swap.Creator, swap.Owner) && swap.Token == swap.From:
		for _, asset := range swap.Assets {
			if err = bc.TokenBalanceAddWithTicker(types.AddrFromBytes(swap.Owner), new(big.Int).SetBytes(asset.Amount), asset.Group, "multi-swap cancel"); err != nil {
				return err
			}
		}
	case bytes.Equal(swap.Creator, swap.Owner) && swap.Token == swap.To:
		if err = bc.AllowedIndustrialBalanceAdd(types.AddrFromBytes(swap.Owner), swap.Assets, "reverse multi-swap cancel"); err != nil {
			return err
		}
	case bytes.Equal(swap.Creator, []byte("0000")) && swap.Token == swap.To:
		for _, asset := range swap.Assets {
			if err = balance.Add(bc.GetStub(), balance.BalanceTypeGiven, swap.From, "", new(mathbig.Int).SetBytes(asset.Amount)); err != nil {
				return err
			}
		}
	}

	return MultiSwapDel(bc.GetStub(), swapID)
}

// MultiSwapLoad - loads multiswap from the ledger
func MultiSwapLoad(stub shim.ChaincodeStubInterface, swapID string) (*proto.MultiSwap, error) {
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

// MultiSwapSave - saves multiswap to the ledger
func MultiSwapSave(stub shim.ChaincodeStubInterface, swapID string, swap *proto.MultiSwap) error {
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

// MultiSwapDel - deletes multiswap from the ledger
func MultiSwapDel(stub shim.ChaincodeStubInterface, swapID string) error {
	key, err := stub.CreateCompositeKey(MultiSwapCompositeType, []string{swapID})
	if err != nil {
		return err
	}
	return stub.DelState(key)
}

// QueryGroupBalanceOf - returns balance of the token for user address
func (bc *BaseContract) QueryGroupBalanceOf(address *types.Address) (map[string]string, error) {
	return bc.IndustrialBalanceGet(address)
}
