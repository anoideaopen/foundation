package mock

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/keys"
	"github.com/anoideaopen/foundation/keys/eth"
	"github.com/anoideaopen/foundation/mock/stub"
	"github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcd/btcutil/base58"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

const (
	batchRobotCert = "0a0a61746f6d797a654d535012d7062d2d2d2d2d42" +
		"4547494e2043455254494649434154452d2d2d2d2d0a4d494943536" +
		"a434341664367417749424167495241496b514e37444f456b683668" +
		"6f52425057633157495577436759494b6f5a497a6a3045417749776" +
		"75963780a437a414a42674e5642415954416c56544d524d77455159" +
		"445651514945777044595778705a6d3979626d6c684d52597746415" +
		"9445651514845773154595734670a526e4a68626d4e7063324e764d" +
		"534d77495159445651514b45787068644739746558706c4c6e56686" +
		"443356b624851755958527662586c365a53356a6144456d0a4d4351" +
		"474131554541784d64593245755958527662586c365a53353159585" +
		"1755a4778304c6d463062323135656d5575593267774868634e4d6a" +
		"41784d44457a0a4d4467314e6a41775768634e4d7a41784d4445784" +
		"d4467314e6a4177576a42324d517377435159445651514745774a56" +
		"557a45544d4245474131554543424d4b0a5132467361575a76636d3" +
		"570595445574d4251474131554542784d4e5532467549455a795957" +
		"356a61584e6a627a45504d4130474131554543784d47593278700a5" +
		"a5735304d536b774a7759445651514444434256633256794d554268" +
		"644739746558706c4c6e56686443356b624851755958527662586c3" +
		"65a53356a6144425a0a4d424d4742797147534d3439416745474343" +
		"7147534d3439417745484130494142427266315057484d51674d736" +
		"e786263465a346f3579774b476e677830594e0a504b627049433542" +
		"3761446f6a46747932576e4871416b5656723270697853502b46684" +
		"97634434c634935633162473963365a375738616a5454424c4d4134" +
		"470a41315564447745422f775145417749486744414d42674e56485" +
		"24d4241663845416a41414d437347413155644977516b4d434b4149" +
		"464b2f5335356c6f4865700a6137384441363173364e6f7433727a4" +
		"367436f435356386f71462b37585172344d416f4743437147534d34" +
		"3942414d43413067414d4555434951436e6870476d0a58515664754" +
		"b632b634266554d6b31494a6835354444726b3335436d436c4d6570" +
		"41533353674967596b634d6e5a6b385a42727179796953544d64665" +
		"26248740a5a32506837364e656d536b62345651706230553d0a2d2d" +
		"2d2d2d454e442043455254494649434154452d2d2d2d2d0a"
	userCert = `MIICSTCCAe+gAwIBAgIQW3KyKC2acfVxSNneRkHZPjAKBggqhkjOPQQDAjCBhzEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xIzAhBgNVBAoTGmF0b215emUudWF0LmRsdC5hdG9teXplLmNoMSYw
JAYDVQQDEx1jYS5hdG9teXplLnVhdC5kbHQuYXRvbXl6ZS5jaDAeFw0yMDEwMTMw
ODU2MDBaFw0zMDEwMTEwODU2MDBaMHYxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMQ8wDQYDVQQLEwZjbGll
bnQxKTAnBgNVBAMMIFVzZXI5QGF0b215emUudWF0LmRsdC5hdG9teXplLmNoMFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEp5H9GVCTmUnVo8dHBTCT7cHmK4xn2X+Y
jJEsrbhodUt9GjUx04uOo05uRWhOI+O4fi0EEu+RSkx98hFUapWfRqNNMEswDgYD
VR0PAQH/BAQDAgeAMAwGA1UdEwEB/wQCMAAwKwYDVR0jBCQwIoAgUr9LnmWgd6lr
vwMDrWzo2i3evMKAKgJJXyioX7tdCvgwCgYIKoZIzj0EAwIDSAAwRQIhAPUozDTR
MOS4WBh87DbsJjI8gIuXPGXwoFXDQQhc2gz0AiAz9jt95z3MKnwj0dWPhjnzAGP8
8PrsVxYtGp6/TnpiPQ==`
)

const (
	shouldNotBeHereMsg = "shouldn't be here"
)

// Wallet is a wallet
type Wallet struct {
	ledger *Ledger

	*keys.Keys

	addr          string
	addrSecp256k1 string
	addrGOST      string
}

// NewWallet creates new wallet
func (l *Ledger) NewWallet() *Wallet {
	keysStr, err := keys.GenerateAllKeys()
	require.NoError(l.t, err)

	hash := sha3.Sum256(keysStr.PublicKeyEd25519)
	hashSecp256k1 := sha3.Sum256(eth.PublicKeyBytes(keysStr.PublicKeySecp256k1))
	hashGOST := sha3.Sum256(keysStr.PublicKeyGOST.Raw())

	return &Wallet{
		ledger:        l,
		Keys:          keysStr,
		addr:          base58.CheckEncode(hash[1:], hash[0]),
		addrGOST:      base58.CheckEncode(hashGOST[1:], hashGOST[0]),
		addrSecp256k1: base58.CheckEncode(hashSecp256k1[1:], hashSecp256k1[0]),
	}
}

// NewMultisigWallet creates new multisig wallet
func (l *Ledger) NewMultisigWallet(n int) *Multisig {
	wlt := &Multisig{Wallet: Wallet{ledger: l}}
	for i := 0; i < n; i++ {
		pKey, sKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(l.t, err)
		wlt.pKeys = append(wlt.pKeys, pKey)
		wlt.sKeys = append(wlt.sKeys, sKey)
	}

	binPubKeys := make([][]byte, len(wlt.pKeys))
	for i, k := range wlt.pKeys {
		binPubKeys[i] = k
	}
	sort.Slice(binPubKeys, func(i, j int) bool {
		return bytes.Compare(binPubKeys[i], binPubKeys[j]) < 0
	})

	hashedAddr := sha3.Sum256(bytes.Join(binPubKeys, []byte("")))
	wlt.addr = base58.CheckEncode(hashedAddr[1:], hashedAddr[0])
	return wlt
}

// NewWalletFromKey creates new wallet from key
func (l *Ledger) NewWalletFromKey(key string) *Wallet {
	keysStr, err := keys.GenerateEd25519FromBase58(key)
	require.NoError(l.t, err)
	hash := sha3.Sum256(keysStr.PublicKeyEd25519)
	return &Wallet{
		ledger: l,
		Keys:   keysStr,
		addr:   base58.CheckEncode(hash[1:], hash[0]),
	}
}

// NewWalletFromHexKey creates new wallet from hex key
func (l *Ledger) NewWalletFromHexKey(key string) *Wallet {
	keysStr, err := keys.GenerateEd25519FromHex(key)
	require.NoError(l.t, err)
	hash := sha3.Sum256(keysStr.PublicKeyEd25519)
	return &Wallet{
		ledger: l,
		Keys:   keysStr,
		addr:   base58.CheckEncode(hash[1:], hash[0]),
	}
}

func getWalletKeyType(stub shim.ChaincodeStubInterface, address string) proto.KeyType {
	ck, err := stub.CreateCompositeKey("pk_type", []string{address})
	if err != nil {
		panic(err)
	}
	raw, err := stub.GetState(ck)
	if err != nil {
		panic(err)
	}
	return proto.KeyType(proto.KeyType_value[string(raw)])
}

func (w *Wallet) saveKeyType() {
	const (
		stubACLName      = "acl"
		compositeKeyType = "pk_type"
	)
	stubACL, ok := w.ledger.stubs[stubACLName]
	if !ok {
		panic("stub not found")
	}
	txID := fmt.Sprintf("%s_%s", w.addr, w.KeyType.String())
	stubACL.MockTransactionStart(txID)
	address := w.addr
	switch w.KeyType {
	case proto.KeyType_secp256k1:
		address = w.addrSecp256k1
	case proto.KeyType_gost:
		address = w.addrGOST
	}
	compositeKey, err := stubACL.CreateCompositeKey(compositeKeyType, []string{address})
	if err != nil {
		panic(err)
	}
	if err = stubACL.PutState(compositeKey, []byte(w.KeyType.String())); err != nil {
		panic(err)
	}
	stubACL.MockTransactionEnd(txID)
}

func (w *Wallet) UseSecp256k1Key() {
	w.KeyType = proto.KeyType_secp256k1
	w.saveKeyType()
}

func (w *Wallet) UseGOSTKey() {
	w.KeyType = proto.KeyType_gost
	w.saveKeyType()
}

// ChangeKeys change private key, then public key will be derived and changed too
func (w *Wallet) ChangeKeys(sKey ed25519.PrivateKey) error {
	w.PrivateKeyEd25519 = sKey
	var ok bool
	w.PublicKeyEd25519, ok = sKey.Public().(ed25519.PublicKey)
	if !ok {
		return errors.New("failed to derive public key from secret")
	}
	return nil
}

// Address returns the address of the wallet
func (w *Wallet) Address() string {
	switch w.KeyType {
	case proto.KeyType_gost:
		return w.addrGOST
	case proto.KeyType_secp256k1:
		return w.addrSecp256k1
	default:
		return w.addr
	}
}

// PubKey returns the public key of the wallet
func (w *Wallet) PubKey() []byte {
	return w.PublicKeyEd25519
}

// SecretKey returns the secret key of the wallet
func (w *Wallet) SecretKey() []byte {
	return w.PrivateKeyEd25519
}

// SetPubKey sets the public key of the wallet
func (w *Wallet) SetPubKey(pk ed25519.PublicKey) {
	w.PublicKeyEd25519 = pk
}

// AddressType returns the address type of the wallet
func (w *Wallet) AddressType() *types.Address {
	value, ver, err := base58.CheckDecode(w.addr)
	if err != nil {
		panic(err)
	}
	return &types.Address{Address: append([]byte{ver}, value...)[:32]}
}

func (w *Wallet) addBalance(stub *stub.Stub, amount *big.Int, balanceType balance.BalanceType, path ...string) {
	key, err := stub.CreateCompositeKey(balanceType.String(), append([]string{w.Address()}, path...))
	require.NoError(w.ledger.t, err)
	data := stub.State[key]
	bal := new(big.Int).SetBytes(data)
	newBalance := new(big.Int).Add(bal, amount)
	_ = stub.PutBalanceToState(key, newBalance)
}

// CheckGivenBalanceShouldBe checks the balance of the wallet
func (w *Wallet) CheckGivenBalanceShouldBe(ch string, token string, expectedBalance uint64) {
	st := w.ledger.stubs[ch]
	key, err := st.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{token})
	require.NoError(w.ledger.t, err)
	rawRecord := st.State[key]
	if rawRecord == nil && expectedBalance == 0 {
		return
	}
	actualBalanceInt := new(big.Int).SetBytes(rawRecord)
	expectedBalanceInt := new(big.Int).SetUint64(expectedBalance)
	require.Equal(w.ledger.t, expectedBalanceInt, actualBalanceInt)
}

// AddBalance adds balance to the wallet
func (w *Wallet) AddBalance(ch string, amount uint64) {
	w.addBalance(w.ledger.stubs[ch], new(big.Int).SetUint64(amount), balance.BalanceTypeToken)
}

// AddAllowedBalance adds allowed balance to the wallet
func (w *Wallet) AddAllowedBalance(ch string, token string, amount uint64) {
	w.addBalance(w.ledger.stubs[ch], new(big.Int).SetUint64(amount), balance.BalanceTypeAllowed, token)
}

// AddGivenBalance adds given balance to the wallet
func (w *Wallet) AddGivenBalance(ch string, givenBalanceChannel string, amount uint64) {
	st := w.ledger.stubs[ch]
	key, err := st.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{givenBalanceChannel})
	require.NoError(w.ledger.t, err)
	newBalance := new(big.Int).SetUint64(amount)
	_ = st.PutBalanceToState(key, newBalance)
}

// AddTokenBalance adds token balance to the wallet
func (w *Wallet) AddTokenBalance(ch string, token string, amount uint64) {
	parts := strings.Split(token, "_")
	w.addBalance(w.ledger.stubs[ch], new(big.Int).SetUint64(amount), balance.BalanceTypeToken, parts[len(parts)-1])
}

// BalanceShouldBe checks the balance of the wallet
func (w *Wallet) BalanceShouldBe(ch string, expected uint64) {
	require.Equal(w.ledger.t, "\""+strconv.FormatUint(expected, 10)+"\"", w.Invoke(ch, "balanceOf", w.Address()))
}

// AllowedBalanceShouldBe checks the allowed balance of the wallet
func (w *Wallet) AllowedBalanceShouldBe(ch string, token string, expected uint64) {
	require.Equal(w.ledger.t, "\""+strconv.FormatUint(expected, 10)+"\"", w.Invoke(ch, "allowedBalanceOf", w.Address(), token))
}

// GivenBalanceShouldBe checks the given balance of the channel
func (w *Wallet) GivenBalanceShouldBe(ch string, token string, expected uint64) {
	require.Equal(w.ledger.t, "\""+strconv.FormatUint(expected, 10)+"\"", w.Invoke(ch, "givenBalance", token))
}

// OtfBalanceShouldBe checks the otf balance of the wallet
func (w *Wallet) OtfBalanceShouldBe(ch string, token string, expected uint64) {
	require.Equal(w.ledger.t, "\""+strconv.FormatUint(expected, 10)+"\"", w.Invoke(ch, "getBalance", w.Address(), token))
}

// IndustrialBalanceShouldBe checks the industrial balance of the wallet
func (w *Wallet) IndustrialBalanceShouldBe(ch, group string, expected uint64) {
	var balances map[string]string
	res := w.Invoke(ch, "industrialBalanceOf", w.Address())
	require.NoError(w.ledger.t, json.Unmarshal([]byte(res), &balances))

	if bal, ok := balances[group]; ok {
		require.Equal(w.ledger.t, strconv.FormatUint(expected, 10), bal)
		return
	}
	if expected == 0 {
		return
	}
	require.Fail(w.ledger.t, "group not found")
}

// GroupBalanceShouldBe checks the group balance of the wallet
func (w *Wallet) GroupBalanceShouldBe(ch, group string, expected uint64) {
	var balances map[string]string
	res := w.Invoke(ch, "groupBalanceOf", w.Address())
	require.NoError(w.ledger.t, json.Unmarshal([]byte(res), &balances))

	if bal, ok := balances[group]; ok {
		require.Equal(w.ledger.t, strconv.FormatUint(expected, 10), bal)
		return
	}
	if expected == 0 {
		return
	}
	require.Fail(w.ledger.t, "group not found")
}

// Invoke invokes a function on the ledger
func (w *Wallet) Invoke(ch, fn string, args ...string) string {
	return w.ledger.doInvoke(ch, txIDGen(), fn, args...)
}

// InvokeReturnsTxID invokes a function on the ledger and returns the transaction ID
func (w *Wallet) InvokeReturnsTxID(ch, fn string, args ...string) string {
	txID := txIDGen()
	w.ledger.doInvoke(ch, txID, fn, args...)
	return txID
}

// InvokeWithError invokes a function on the ledger and returns an error
func (w *Wallet) InvokeWithError(ch, fn string, args ...string) error {
	return w.ledger.doInvokeWithErrorReturned(ch, txIDGen(), fn, args...)
}

func (w *Wallet) InvokeWithPeerResponse(ch, fn string, args ...string) (peer.Response, error) {
	return w.ledger.doInvokeWithPeerResponse(ch, txIDGen(), fn, args...)
}

// SignArgs signs the arguments
func (w *Wallet) SignArgs(ch, fn string, args ...string) []string {
	resp, _ := w.sign(fn, ch, args...)
	return resp
}

func (w *Wallet) WithNonceSignArgs(ch, fn string, nonce string, args ...string) []string {
	resp, _ := w.signWithNonce(fn, ch, nonce, args...)
	return resp
}

// BatchedInvoke invokes a function on the ledger
func (w *Wallet) BatchedInvoke(ch, fn string, args ...string) (string, TxResponse) {
	if err := w.verifyIncoming(ch, fn); err != nil {
		require.NoError(w.ledger.t, err)
		return "", TxResponse{}
	}
	txID := txIDGen()
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
				evts := make(map[string][]byte)
				for _, evt := range ev.GetEvents() {
					evts[evt.GetName()] = evt.GetValue()
				}
				er := ""
				if ev.GetError() != nil {
					er = ev.GetError().GetError()
				}
				return txID, TxResponse{
					Method: ev.GetMethod(),
					Error:  er,
					Result: string(ev.GetResult()),
					Events: evts,
				}
			}
		}
	}
	require.Fail(w.ledger.t, shouldNotBeHereMsg)
	return txID, TxResponse{}
}

func (w *Wallet) publicKeyBytes() []byte {
	switch w.KeyType {
	case proto.KeyType_gost:
		return w.PublicKeyGOST.Raw()
	case proto.KeyType_secp256k1:
		return eth.PublicKeyBytes(w.PublicKeySecp256k1)
	default:
		return w.PublicKeyEd25519
	}
}

func (w *Wallet) sign(fn, ch string, args ...string) ([]string, string) {
	// Artificial delay to update the nonce value.
	time.Sleep(time.Millisecond * 5)

	// Generation of nonce based on current time in milliseconds.
	nonce := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)

	return w.signWithNonce(fn, ch, nonce, args...)
}

func (w *Wallet) signWithNonce(fn, ch string, nonce string, args ...string) ([]string, string) {
	// Forming a message for signature, including function name,
	// empty string (placeholder), channel name, arguments and nonce.
	publicKey := w.publicKeyBytes()

	messageChunks := []string{fn, "", ch, ch}
	messageChunks = append(messageChunks, args...)                  // Adding call arguments.
	messageChunks = append(messageChunks, nonce)                    // Adding nonce.
	messageChunks = append(messageChunks, base58.Encode(publicKey)) // Adding an encoded public key.
	message := []byte(strings.Join(messageChunks, ""))

	// Calculating the hash of the message and signing the hash with the secret key and adding the signature to the message.
	digest, signature, err := keys.SignMessageByKeyType(w.KeyType, w.Keys, message)
	require.NoError(w.ledger.t, err)

	// We remove the function name from the message and add a caption.
	signedMessage := append(messageChunks[1:], base58.Encode(signature)) //nolint:gocritic

	// Return the signed message and hash in hexadecimal format.
	return signedMessage, hex.EncodeToString(digest)
}

// BatchTxResponse is a batch transaction response
type BatchTxResponse map[string]*proto.TxResponse

// DoBatch does a batch transaction
func (w *Wallet) DoBatch(ch string, txID ...string) BatchTxResponse {
	if err := w.verifyIncoming(ch, "fn"); err != nil {
		require.NoError(w.ledger.t, err)
		return BatchTxResponse{}
	}
	b := &proto.Batch{}
	for _, id := range txID {
		x, err := hex.DecodeString(id)
		require.NoError(w.ledger.t, err)
		b.TxIDs = append(b.TxIDs, x)
	}
	data, err := pb.Marshal(b)
	require.NoError(w.ledger.t, err)

	cert, err := hex.DecodeString(batchRobotCert)
	require.NoError(w.ledger.t, err)
	w.ledger.stubs[ch].SetCreator(cert)
	res := w.Invoke(ch, core.BatchExecute, string(data))
	out := &proto.BatchResponse{}
	require.NoError(w.ledger.t, pb.Unmarshal([]byte(res), out))

	result := make(BatchTxResponse)
	for _, resp := range out.GetTxResponses() {
		if resp != nil {
			result[hex.EncodeToString(resp.GetId())] = resp
		}
	}
	return result
}

// TxHasNoError checks if the transaction has no error
func (br BatchTxResponse) TxHasNoError(t *testing.T, txID ...string) {
	for _, id := range txID {
		res, ok := br[id]
		require.True(t, ok, "tx %s doesn't exist in batch response", id)
		if !ok {
			return
		}
		msg := ""
		if res.GetError() != nil {
			msg = res.GetError().GetError()
		}
		require.Nil(t, res.GetError(), msg)
	}
}

// RawSignedInvoke invokes a function on the ledger
func (w *Wallet) RawSignedInvoke(ch string, fn string, args ...string) (string, TxResponse, []*proto.Swap) {
	invoke, response, swaps, _ := w.RawSignedMultiSwapInvoke(ch, fn, args...)
	return invoke, response, swaps
}

// Ledger returns the ledger
func (w *Wallet) Ledger() *Ledger {
	return w.ledger
}

// RawSignedMultiSwapInvoke invokes a function on the ledger
func (w *Wallet) RawSignedMultiSwapInvoke(ch, fn string, args ...string) (string, TxResponse, []*proto.Swap, []*proto.MultiSwap) {
	if err := w.verifyIncoming(ch, fn); err != nil {
		require.NoError(w.ledger.t, err)
		return "", TxResponse{}, nil, nil
	}
	txID := txIDGen()
	args, _ = w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	require.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("platformMSP", cert)
	w.ledger.doInvoke(ch, txID, fn, args...)

	id, err := hex.DecodeString(txID)
	require.NoError(w.ledger.t, err)
	data, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{id}})
	require.NoError(w.ledger.t, err)

	cert, err = hex.DecodeString(batchRobotCert)
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
				evts := make(map[string][]byte)
				for _, evt := range ev.GetEvents() {
					evts[evt.GetName()] = evt.GetValue()
				}
				er := ""
				if ev.GetError() != nil {
					er = ev.GetError().GetError()
				}
				return txID, TxResponse{
					Method: ev.GetMethod(),
					Error:  er,
					Result: string(ev.GetResult()),
					Events: evts,
				}, out.GetCreatedSwaps(), out.GetCreatedMultiSwap()
			}
		}
	}
	require.Fail(w.ledger.t, shouldNotBeHereMsg)
	return txID, TxResponse{}, out.GetCreatedSwaps(), out.GetCreatedMultiSwap()
}

// RawSignedInvokeWithErrorReturned invokes a function on the ledger
func (w *Wallet) RawSignedInvokeWithErrorReturned(ch, fn string, args ...string) error {
	if err := w.verifyIncoming(ch, fn); err != nil {
		return err
	}
	txID := txIDGen()
	args, _ = w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	require.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("platformMSP", cert)
	err = w.ledger.doInvokeWithErrorReturned(ch, txID, fn, args...)
	if err != nil {
		return err
	}

	id, err := hex.DecodeString(txID)
	if err != nil {
		return err
	}
	data, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{id}})
	if err != nil {
		return err
	}

	cert, err = hex.DecodeString(batchRobotCert)
	if err != nil {
		return err
	}
	w.ledger.stubs[ch].SetCreator(cert)
	res := w.Invoke(ch, core.BatchExecute, string(data))
	out := &proto.BatchResponse{}
	err = pb.Unmarshal([]byte(res), out)
	if err != nil {
		return err
	}

	e := <-w.ledger.stubs[ch].ChaincodeEventsChannel
	if e.GetEventName() == core.BatchExecute {
		events := &proto.BatchEvent{}
		err = pb.Unmarshal(e.GetPayload(), events)
		if err != nil {
			return err
		}
		for _, ev := range events.GetEvents() {
			if hex.EncodeToString(ev.GetId()) == txID {
				evts := make(map[string][]byte)
				for _, evt := range ev.GetEvents() {
					evts[evt.GetName()] = evt.GetValue()
				}
				if ev.GetError() != nil {
					return errors.New(ev.GetError().GetError())
				}
				return nil
			}
		}
	}
	require.Fail(w.ledger.t, shouldNotBeHereMsg)
	return nil
}

// RawChTransferInvoke invokes a function on the ledger
func (w *Wallet) RawChTransferInvoke(ch, fn string, args ...string) (string, TxResponse, error) {
	if err := w.verifyIncoming(ch, fn); err != nil {
		return "", TxResponse{}, err
	}
	txID := txIDGen()
	cert, err := hex.DecodeString(batchRobotCert)
	require.NoError(w.ledger.t, err)
	w.ledger.stubs[ch].SetCreator(cert)
	err = w.ledger.doInvokeWithErrorReturned(ch, txID, fn, args...)
	if err != nil {
		return "", TxResponse{}, err
	}

	return txID, TxResponse{}, nil
}

// RawChTransferInvokeWithBatch invokes a function on the ledger
func (w *Wallet) RawChTransferInvokeWithBatch(ch string, fn string, args ...string) (string, TxResponse, error) {
	txID, _, err := w.RawChTransferInvoke(ch, fn, args...)
	if err != nil {
		return "", TxResponse{}, err
	}

	id, err := hex.DecodeString(txID)
	if err != nil {
		return "", TxResponse{}, err
	}
	data, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{id}})
	if err != nil {
		return "", TxResponse{}, err
	}

	cert, err := hex.DecodeString(batchRobotCert)
	if err != nil {
		return "", TxResponse{}, err
	}
	w.ledger.stubs[ch].SetCreator(cert)
	res := w.Invoke(ch, core.BatchExecute, string(data))
	out := &proto.BatchResponse{}
	err = pb.Unmarshal([]byte(res), out)
	if err != nil {
		return "", TxResponse{}, err
	}

	e := <-w.ledger.stubs[ch].ChaincodeEventsChannel
	if e.GetEventName() == core.BatchExecute {
		events := &proto.BatchEvent{}
		err = pb.Unmarshal(e.GetPayload(), events)
		if err != nil {
			return "", TxResponse{}, err
		}
		for _, ev := range events.GetEvents() {
			if hex.EncodeToString(ev.GetId()) == txID {
				evts := make(map[string][]byte)
				for _, evt := range ev.GetEvents() {
					evts[evt.GetName()] = evt.GetValue()
				}
				er := ""
				var er1 error
				if ev.GetError() != nil {
					er = ev.GetError().GetError()
					er1 = errors.New(er)
				}
				return txID, TxResponse{
					Method: ev.GetMethod(),
					Error:  er,
					Result: string(ev.GetResult()),
					Events: evts,
				}, er1
			}
		}
	}

	return txID, TxResponse{}, nil
}

// SignedInvoke invokes a function on the ledger
func (w *Wallet) SignedInvoke(ch string, fn string, args ...string) string {
	txID, res, swaps := w.RawSignedInvoke(ch, fn, args...)
	require.Equal(w.ledger.t, "", res.Error)
	for _, swap := range swaps {
		x := proto.Batch{Swaps: []*proto.Swap{{
			Id:      swap.GetId(),
			Creator: []byte("0000"),
			Owner:   swap.GetOwner(),
			Token:   swap.GetToken(),
			Amount:  swap.GetAmount(),
			From:    swap.GetFrom(),
			To:      swap.GetTo(),
			Hash:    swap.GetHash(),
			Timeout: swap.GetTimeout(),
		}}}
		data, err := pb.Marshal(&x)
		require.NoError(w.ledger.t, err)
		cert, err := hex.DecodeString(batchRobotCert)
		require.NoError(w.ledger.t, err)
		w.ledger.stubs[strings.ToLower(swap.GetTo())].SetCreator(cert)
		w.Invoke(strings.ToLower(swap.GetTo()), core.BatchExecute, string(data))
	}
	return txID
}

// SignedMultiSwapsInvoke invokes a function on the ledger
func (w *Wallet) SignedMultiSwapsInvoke(ch string, fn string, args ...string) string {
	txID, res, _, multiSwaps := w.RawSignedMultiSwapInvoke(ch, fn, args...)
	require.Equal(w.ledger.t, "", res.Error)
	for _, swap := range multiSwaps {
		x := proto.Batch{
			MultiSwaps: []*proto.MultiSwap{
				{
					Id:      swap.GetId(),
					Creator: []byte("0000"),
					Owner:   swap.GetOwner(),
					Token:   swap.GetToken(),
					Assets:  swap.GetAssets(),
					From:    swap.GetFrom(),
					To:      swap.GetTo(),
					Hash:    swap.GetHash(),
					Timeout: swap.GetTimeout(),
				},
			},
		}
		data, err := pb.Marshal(&x)
		require.NoError(w.ledger.t, err)
		cert, err := hex.DecodeString(batchRobotCert)
		require.NoError(w.ledger.t, err)
		w.ledger.stubs[swap.GetTo()].SetCreator(cert)
		w.Invoke(swap.GetTo(), core.BatchExecute, string(data))
	}
	return txID
}

// OtfNbInvoke executes non-batched transactions
//
// Deprecated: use NbInvoke instead
func (w *Wallet) OtfNbInvoke(ch string, fn string, args ...string) (string, string) {
	return w.NbInvoke(ch, fn, args...)
}

// NbInvoke executes non-batched transactions
func (w *Wallet) NbInvoke(ch string, fn string, args ...string) (string, string) {
	if err := w.verifyIncoming(ch, fn); err != nil {
		require.NoError(w.ledger.t, err)
		return "", ""
	}
	txID := txIDGen()
	message, hash := w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	require.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("platformMSP", cert)
	w.ledger.doInvoke(ch, txID, fn, message...)

	nested, err := pb.Marshal(&proto.Nested{Args: append([]string{fn}, message...)})
	require.NoError(w.ledger.t, err)

	return base58.Encode(nested), hash
}

func (w *Wallet) verifyIncoming(ch string, fn string) error {
	if ch == "" {
		return errors.New("channel undefined")
	}
	if fn == "" {
		return errors.New("chaincode method undefined")
	}
	if _, ok := w.ledger.stubs[ch]; !ok {
		return fmt.Errorf("stub of [%s] not found", ch)
	}

	return nil
}
