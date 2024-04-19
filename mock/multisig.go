package mock

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

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
	require.NoError(w.ledger.t, err)
	data, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{id}})
	require.NoError(w.ledger.t, err)

	cert, err := hex.DecodeString(batchRobotCert)
	require.NoError(w.ledger.t, err)
	w.ledger.stubs[ch].SetCreator(cert)
	res := w.Invoke(ch, core.BatchExecute, string(data))
	out := &proto.BatchResponse{}
	require.NoError(w.ledger.t, pb.Unmarshal([]byte(res), out))

	e := <-w.ledger.stubs[ch].ChaincodeEventsChannel
	if e.GetEventName() == core.BatchExecute {
		events := &proto.BatchEvent{}
		require.NoError(w.ledger.t, pb.Unmarshal(e.GetPayload(), events))
		for _, ev := range events.GetEvents() {
			if hex.EncodeToString(ev.GetId()) == txID {
				events1 := make(map[string][]byte)
				for _, evt := range ev.GetEvents() {
					events1[evt.GetName()] = evt.GetValue()
				}
				err := ""
				if ev.GetError() != nil {
					err = ev.GetError().GetError()
				}
				return txID, TxResponse{
					Method: ev.GetMethod(),
					Error:  err,
					Result: string(ev.GetResult()),
					Events: events1,
				}, out.GetCreatedSwaps()
			}
		}
	}
	require.Fail(w.ledger.t, shouldNotBeHereMsg)
	return txID, TxResponse{}, out.GetCreatedSwaps()
}

// SecretKeys returns private keys of multisig wallet
func (w *Multisig) SecretKeys() []ed25519.PrivateKey {
	return w.sKeys
}

// PubKeys returns public keys of multisig wallet
func (w *Multisig) PubKeys() []ed25519.PublicKey {
	return w.pKeys
}
