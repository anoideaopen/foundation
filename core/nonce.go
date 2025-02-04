package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"google.golang.org/protobuf/proto"
)

const StateKeyNonce byte = 42 // hex: 2a

const (
	doublingMemoryCoef = 2
	LeftBorderNonce    = 1e12
	RightBorderNonce   = 1e13
	// defaultNonceTTL is time for nonce. If attempting to execute a transaction in a batch
	// that is older than the maximum nonce (at the current moment) by more than NonceTTL,
	// we will not execute it and return an error.
	defaultNonceTTL               = 50 * time.Second
	defaultNonceTTLForCreateCCTTo = 3 * time.Hour
	multiKoeffForGeneralNonce     = 1e6
)

type Nonce struct {
	ttl        time.Duration
	guardF     func(uint64) error
	transformF func(uint64) uint64
}

func (n *Nonce) check(
	stub shim.ChaincodeStubInterface,
	sender *types.Sender,
	nonce uint64,
	args ...string,
) error {
	var attributes []string

	noncePrefix := hex.EncodeToString([]byte{StateKeyNonce})
	attributes = append(append(attributes, args...), sender.Address().String())
	nonceKey, err := stub.CreateCompositeKey(noncePrefix, attributes)
	if err != nil {
		return err
	}
	data, err := stub.GetState(nonceKey)
	if err != nil {
		return err
	}

	lastNonce := new(pb.Nonce)
	if len(data) > 0 {
		if err = proto.Unmarshal(data, lastNonce); err != nil {
			log := logger.Logger()
			log.Warningf("error unmarshal nonce, maybe old nonce. error: %v", err)
			// let's just say it's an old nonse
			lastNonce.Nonce = []uint64{new(big.Int).SetBytes(data).Uint64()}
		}
	}

	nonceTTL := n.ttl
	if nonceTTL == 0 {
		nonceTTL = defaultNonceTTL
	}

	// guard function
	if n.guardF == nil {
		n.guardF = func(nnc uint64) error {
			// to support common (old) nones multiply and divide by a factor
			if nnc < LeftBorderNonce || nnc >= RightBorderNonce {
				return errors.New("incorrect nonce format")
			}
			return nil
		}
	}
	if err = n.guardF(nonce); err != nil {
		return err
	}

	// transform func
	if n.transformF == nil {
		// to support common (old) nones multiply and divide by a factor
		n.transformF = func(u uint64) uint64 {
			return u * multiKoeffForGeneralNonce
		}
	}
	nonce = n.transformF(nonce)

	lastNonce.Nonce, err = n.set(nonce, lastNonce.GetNonce(), nonceTTL)
	if err != nil {
		return err
	}

	data, err = proto.Marshal(lastNonce)
	if err != nil {
		return err
	}

	return stub.PutState(nonceKey, data)
}

func (n *Nonce) set(nonce uint64, lastNonce []uint64, nonceTTL time.Duration) ([]uint64, error) {
	if len(lastNonce) == 0 {
		return []uint64{nonce}, nil
	}

	l := len(lastNonce)

	last := lastNonce[l-1]

	if nonce > last {
		lastNonce = append(lastNonce, nonce)
		l = len(lastNonce)
		last = lastNonce[l-1]

		index := sort.Search(l, func(i int) bool { return last-lastNonce[i] <= uint64(nonceTTL.Nanoseconds()) })
		return lastNonce[index:], nil
	}

	if last-nonce > uint64(nonceTTL.Nanoseconds()) {
		return lastNonce, fmt.Errorf("incorrect nonce %d, less than %d", nonce, last)
	}

	index := sort.Search(l, func(i int) bool { return lastNonce[i] >= nonce })
	if index != l && lastNonce[index] == nonce {
		return lastNonce, fmt.Errorf("nonce %d already exists", nonce)
	}

	// paste
	if cap(lastNonce) > len(lastNonce) {
		lastNonce = lastNonce[:len(lastNonce)+1]
		copy(lastNonce[index+1:], lastNonce[index:])
		lastNonce[index] = nonce
	} else {
		x := make([]uint64, 0, len(lastNonce)*doublingMemoryCoef)
		x = append(x, lastNonce[:index]...)
		x = append(x, nonce)
		x = append(x, lastNonce[index:]...)
		lastNonce = x
	}

	return lastNonce, nil
}
