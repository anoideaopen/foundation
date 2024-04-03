package mock

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"
)

const batchExecute = "batchExecute"

// Multisig is a mock for multisig wallet
type Multisig struct {
	Wallet
	pKeys []ed25519.PublicKey
	sKeys []ed25519.PrivateKey
}

// Address returns address of multisig wallet
func (w *Multisig) Address() string {
	return w.addr
}

// AddressType returns address of multisig wallet
func (w *Multisig) AddressType() *types.Address {
	value, ver, err := base58.CheckDecode(w.addr)
	if err != nil {
		panic(err)
	}
	return &types.Address{Address: append([]byte{ver}, value...)[:32]}
}

// ChangeKeysFor changes private and public keys for Multisig member with specific index
func (w *Multisig) ChangeKeysFor(index int, sKey ed25519.PrivateKey) error {
	w.sKeys[index] = sKey
	var ok bool
	w.pKeys[index], ok = sKey.Public().(ed25519.PublicKey)
	if !ok {
		return errors.New("failed to derive public key from secret")
	}

	return nil
}

func (w *Multisig) sign(signCnt int, fn string, ch string, args ...string) ([]string, string) {
	time.Sleep(time.Millisecond * 5) //nolint:gomnd
	nonce := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	result := append(append([]string{fn, "", ch, ch}, args...), nonce)
	for _, pk := range w.pKeys {
		result = append(result, base58.Encode(pk))
	}
	message := sha3.Sum256([]byte(strings.Join(result, "")))
	for _, skey := range w.sKeys {
		if signCnt > 0 {
			result = append(result, base58.Encode(ed25519.Sign(skey, message[:])))
		} else {
			result = append(result, "")
		}
		signCnt--
	}

	return result[1:], hex.EncodeToString(message[:])
}

// RawSignedInvoke invokes chaincode function with specific arguments and signs it with multisig wallet
func (w *Multisig) RawSignedInvoke(signCnt int, ch string, fn string, args ...string) (string, TxResponse, []*proto.Swap) {
	txID := txIDGen()
	args, _ = w.sign(signCnt, fn, ch, args...)
	w.ledger.doInvoke(ch, txID, fn, args...)

	id, err := hex.DecodeString(txID)
	assert.NoError(w.ledger.t, err)
	data, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{id}})
	assert.NoError(w.ledger.t, err)

	cert, err := hex.DecodeString(batchRobotCert)
	assert.NoError(w.ledger.t, err)
	w.ledger.stubs[ch].SetCreator(cert)
	res := w.Invoke(ch, batchExecute, string(data))
	out := &proto.BatchResponse{}
	assert.NoError(w.ledger.t, pb.Unmarshal([]byte(res), out))

	e := <-w.ledger.stubs[ch].ChaincodeEventsChannel
	if e.EventName == batchExecute {
		events := &proto.BatchEvent{}
		assert.NoError(w.ledger.t, pb.Unmarshal(e.Payload, events))
		for _, e := range events.Events {
			if hex.EncodeToString(e.Id) == txID {
				events := make(map[string][]byte)
				for _, e := range e.Events {
					events[e.Name] = e.Value
				}
				err := ""
				if e.Error != nil {
					err = e.Error.Error
				}
				return txID, TxResponse{
					Method: e.Method,
					Error:  err,
					Result: string(e.Result),
					Events: events,
				}, out.CreatedSwaps
			}
		}
	}
	assert.Fail(w.ledger.t, shouldNotBeHereMsg)
	return txID, TxResponse{}, out.CreatedSwaps
}

// SecretKeys returns private keys of multisig wallet
func (w *Multisig) SecretKeys() []ed25519.PrivateKey {
	return w.sKeys
}

// PubKeys returns public keys of multisig wallet
func (w *Multisig) PubKeys() []ed25519.PublicKey {
	return w.pKeys
}
