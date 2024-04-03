package core

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

const (
	doublingMemoryCoef    = 2
	lenTimeInMilliseconds = 13
)

func checkNonce(nonceTTL uint, prefix StateKey) NonceCheckFn {
	return func(stub shim.ChaincodeStubInterface, sender *types.Sender, nonce uint64) error {
		noncePrefix := hex.EncodeToString([]byte{byte(prefix)})
		nonceKey, err := stub.CreateCompositeKey(noncePrefix, []string{sender.Address().String()})
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
				logger := Logger()
				logger.Warningf("error unmarshal nonce, maybe old nonce. error: %v", err)
				// let's just say it's an old nonse
				lastNonce.Nonce = []uint64{new(big.Int).SetBytes(data).Uint64()}
			}
		}

		mayBeOtherSorting := false
		if prefix == StateKeyPassedNonce {
			mayBeOtherSorting = true
		}

		lastNonce.Nonce, err = setNonce(nonce, lastNonce.Nonce, nonceTTL, mayBeOtherSorting)
		if err != nil {
			return err
		}

		data, err = proto.Marshal(lastNonce)
		if err != nil {
			return err
		}

		return stub.PutState(nonceKey, data)
	}
}

func setNonce(nonce uint64, lastNonce []uint64, nonceTTL uint, mayBeOtherSorting bool) ([]uint64, error) {
	if len(strconv.FormatUint(nonce, 10)) != lenTimeInMilliseconds {
		return lastNonce, fmt.Errorf("incorrect nonce format")
	}

	if len(lastNonce) == 0 {
		return []uint64{nonce}, nil
	}

	l := len(lastNonce)

	if mayBeOtherSorting && l > 1 && lastNonce[0] > lastNonce[l-1] {
		// for US you initially need to reverse the slice
		sort.Slice(lastNonce, func(i, j int) bool { return lastNonce[i] <= lastNonce[j] })
	}

	last := lastNonce[l-1]

	if nonceTTL == 0 {
		// old check
		if nonce <= last {
			return lastNonce, fmt.Errorf("incorrect nonce, current %d", last)
		}
		return []uint64{nonce}, nil
	}

	ttl := time.Second * time.Duration(nonceTTL)

	if nonce > last {
		lastNonce = append(lastNonce, nonce)
		l = len(lastNonce)
		last = lastNonce[l-1]

		index := sort.Search(l, func(i int) bool { return last-lastNonce[i] <= uint64(ttl.Milliseconds()) })
		return lastNonce[index:], nil
	}

	if last-nonce > uint64(ttl.Milliseconds()) {
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
