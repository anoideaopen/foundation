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

const StateKeyNonce byte = 42 // hex: 2a

const (
	doublingMemoryCoef    = 2
	lenTimeInMilliseconds = 13
	// defaultNonceTTL is time in seconds for nonce. If attempting to execute a transaction in a batch
	// that is older than the maximum nonce (at the current moment) by more than NonceTTL,
	// we will not execute it and return an error.
	defaultNonceTTL = 50
)

func checkNonce(
	stub shim.ChaincodeStubInterface,
	sender *types.Sender,
	nonce uint64,
) error {
	noncePrefix := hex.EncodeToString([]byte{StateKeyNonce})
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

	lastNonce.Nonce, err = setNonce(nonce, lastNonce.Nonce, defaultNonceTTL)
	if err != nil {
		return err
	}

	data, err = proto.Marshal(lastNonce)
	if err != nil {
		return err
	}

	return stub.PutState(nonceKey, data)
}

func setNonce(nonce uint64, lastNonce []uint64, nonceTTL uint) ([]uint64, error) {
	if len(strconv.FormatUint(nonce, 10)) != lenTimeInMilliseconds {
		return lastNonce, fmt.Errorf("incorrect nonce format")
	}

	if len(lastNonce) == 0 {
		return []uint64{nonce}, nil
	}

	l := len(lastNonce)

	last := lastNonce[l-1]

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
