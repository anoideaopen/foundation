package mock

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/cctransfer"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/internal/config"
	"github.com/anoideaopen/foundation/mock/stub"
	"github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ddulesov/gogost/gost3410"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

type Ledger struct {
	t                   *testing.T
	stubs               map[string]*stub.Stub
	keyEvents           map[string]chan *peer.ChaincodeEvent
	txResponseEvents    map[string]chan TxResponse
	txResponseEventLock *sync.Mutex
	batchPrefix         string
}

// GetStubByKey returns stub by key
func (ledger *Ledger) GetStubByKey(key string) *stub.Stub {
	return ledger.stubs[key]
}

// UpdateStubTxID updates stub txID
func (ledger *Ledger) UpdateStubTxID(stubName string, newTxID string) {
	ledger.stubs[stubName].TxID = newTxID
}

// NewLedger creates new ledger
func NewLedger(t *testing.T, options ...string) *Ledger {
	lvl := logrus.ErrorLevel
	var err error
	if level, ok := os.LookupEnv("LOG"); ok {
		lvl, err = logrus.ParseLevel(level)
		assert.NoError(t, err)
	}
	logrus.SetLevel(lvl)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	aclStub := stub.NewMockStub("acl", new(mockACL))
	assert.Equal(t, int32(http.StatusOK), aclStub.MockInit(hex.EncodeToString([]byte("acl")), nil).Status)

	prefix := config.BatchPrefix
	if len(options) != 0 && options[0] != "" {
		prefix = options[0]
	}

	return &Ledger{
		t:                   t,
		stubs:               map[string]*stub.Stub{"acl": aclStub},
		keyEvents:           make(map[string]chan *peer.ChaincodeEvent),
		txResponseEvents:    make(map[string]chan TxResponse),
		txResponseEventLock: &sync.Mutex{},
		batchPrefix:         prefix,
	}
}

// SetACL sets acl stub
func (ledger *Ledger) SetACL(aclStub *stub.Stub) {
	ledger.stubs["acl"] = aclStub
}

// TxResponse returns txResponse event
type TxResponse struct {
	Method     string                    `json:"method"`
	Error      string                    `json:"error,omitempty"`
	Result     string                    `json:"result"`
	Events     map[string][]byte         `json:"events,omitempty"`
	Accounting []*proto.AccountingRecord `json:"accounting"`
}

// NewCCArgsArr
// Deprecated: added only for backward compatibility.
func (ledger *Ledger) NewCCArgsArr(
	name string,
	bci core.BaseContractInterface,
	initArgs []string,
	opts ...core.ChaincodeOption,
) string {
	_, exists := ledger.stubs[name]
	assert.False(
		ledger.t,
		exists,
		fmt.Sprintf("stub with name '%s' has already exist in ledger mock; "+
			"try to use other chaincode name.", name),
	)

	cc, err := core.NewCC(bci, opts...)
	assert.NoError(ledger.t, err)
	ledger.stubs[name] = stub.NewMockStub(name, cc)
	ledger.stubs[name].ChannelID = name

	ledger.stubs[name].MockPeerChaincode("acl/acl", ledger.stubs["acl"])

	err = ledger.stubs[name].SetAdminCreatorCert("platformMSP")
	assert.NoError(ledger.t, err)

	args := make([][]byte, 0, len(initArgs))
	for _, ia := range initArgs {
		args = append(args, []byte(ia))
	}

	res := ledger.stubs[name].MockInit(txIDGen(), args)
	message := res.Message
	if message != "" {
		return message
	}

	ledger.keyEvents[name] = make(chan *peer.ChaincodeEvent, 1)
	return ""
}

func (ledger *Ledger) NewCC(
	name string,
	bci core.BaseContractInterface,
	config string,
	opts ...core.ChaincodeOption,
) string {
	_, exists := ledger.stubs[name]
	assert.False(
		ledger.t,
		exists,
		fmt.Sprintf("stub with name '%s' has already exist in ledger mock; "+
			"try to use other chaincode name.", name),
	)

	cc, err := core.NewCC(bci, opts...)
	assert.NoError(ledger.t, err)
	ledger.stubs[name] = stub.NewMockStub(name, cc)
	ledger.stubs[name].ChannelID = name

	ledger.stubs[name].MockPeerChaincode("acl/acl", ledger.stubs["acl"])

	err = ledger.stubs[name].SetAdminCreatorCert("platformMSP")
	assert.NoError(ledger.t, err)
	res := ledger.stubs[name].MockInit(txIDGen(), [][]byte{[]byte(config)})
	message := res.Message
	if message != "" {
		return message
	}

	ledger.keyEvents[name] = make(chan *peer.ChaincodeEvent, 1)
	return ""
}

// GetStub returns stub
func (ledger *Ledger) GetStub(name string) *stub.Stub {
	return ledger.stubs[name]
}

// WaitMultiSwapAnswer waits for multi swap answer
func (ledger *Ledger) WaitMultiSwapAnswer(name string, id string, timeout time.Duration) {
	interval := time.Second / 2 //nolint:gomnd
	ticker := time.NewTicker(interval)
	count := timeout.Microseconds() / interval.Microseconds()
	key, err := ledger.stubs[name].CreateCompositeKey(core.MultiSwapCompositeType, []string{id})
	assert.NoError(ledger.t, err)
	for count > 0 {
		count--
		<-ticker.C
		if _, exists := ledger.stubs[name].State[key]; exists {
			return
		}
	}
	for k, v := range ledger.stubs[name].State {
		fmt.Println(k, string(v))
	}
	assert.Fail(ledger.t, "timeout exceeded")
}

// WaitSwapAnswer waits for swap answer
func (ledger *Ledger) WaitSwapAnswer(name string, id string, timeout time.Duration) {
	interval := time.Second / 2 //nolint:gomnd
	ticker := time.NewTicker(interval)
	count := timeout.Microseconds() / interval.Microseconds()
	key, err := ledger.stubs[name].CreateCompositeKey("swaps", []string{id})
	assert.NoError(ledger.t, err)
	for count > 0 {
		count--
		<-ticker.C
		if _, exists := ledger.stubs[name].State[key]; exists {
			return
		}
	}
	for k, v := range ledger.stubs[name].State {
		fmt.Println(k, string(v))
	}
	assert.Fail(ledger.t, "timeout exceeded")
}

// WaitChTransferTo waits for transfer to event
func (ledger *Ledger) WaitChTransferTo(name string, id string, timeout time.Duration) {
	interval := time.Second / 2 //nolint:gomnd
	ticker := time.NewTicker(interval)
	count := timeout.Microseconds() / interval.Microseconds()
	key := cctransfer.CCToTransfer(id)
	for count > 0 {
		count--
		<-ticker.C
		if _, exists := ledger.stubs[name].State[key]; exists {
			return
		}
	}
	for k, v := range ledger.stubs[name].State {
		fmt.Println(k, string(v))
	}
	assert.Fail(ledger.t, "timeout exceeded")
}

// NewWallet creates new wallet
func (ledger *Ledger) NewWallet() *Wallet {
	pKey, sKey, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(ledger.t, err)

	sKeyGOST, err := gost3410.GenPrivateKey(
		gost3410.CurveIdGostR34102001CryptoProXchAParamSet(),
		gost3410.Mode2001,
		rand.Reader,
	)
	assert.NoError(ledger.t, err)

	pKeyGOST, err := sKeyGOST.PublicKey()
	assert.NoError(ledger.t, err)

	var (
		hash     = sha3.Sum256(pKey)
		hashGOST = sha3.Sum256(pKeyGOST.Raw())
	)
	return &Wallet{
		ledger:   ledger,
		sKey:     sKey,
		pKey:     pKey,
		sKeyGOST: sKeyGOST,
		pKeyGOST: pKeyGOST,
		addr:     base58.CheckEncode(hash[1:], hash[0]),
		addrGOST: base58.CheckEncode(hashGOST[1:], hashGOST[0]),
	}
}

// NewMultisigWallet creates new multisig wallet
func (ledger *Ledger) NewMultisigWallet(n int) *Multisig {
	wlt := &Multisig{Wallet: Wallet{ledger: ledger}}
	for i := 0; i < n; i++ {
		pKey, sKey, err := ed25519.GenerateKey(rand.Reader)
		assert.NoError(ledger.t, err)
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
func (ledger *Ledger) NewWalletFromKey(key string) *Wallet {
	decoded, ver, err := base58.CheckDecode(key)
	assert.NoError(ledger.t, err)
	sKey := ed25519.PrivateKey(append([]byte{ver}, decoded...))
	pub, ok := sKey.Public().(ed25519.PublicKey)
	assert.True(ledger.t, ok)
	hash := sha3.Sum256(pub)
	return &Wallet{
		ledger: ledger,
		sKey:   sKey,
		pKey:   pub,
		addr:   base58.CheckEncode(hash[1:], hash[0]),
	}
}

// NewWalletFromHexKey creates new wallet from hex key
func (ledger *Ledger) NewWalletFromHexKey(key string) *Wallet {
	decoded, err := hex.DecodeString(key)
	assert.NoError(ledger.t, err)
	sKey := ed25519.PrivateKey(decoded)
	pub, ok := sKey.Public().(ed25519.PublicKey)
	assert.True(ledger.t, ok)
	hash := sha3.Sum256(pub)
	return &Wallet{ledger: ledger, sKey: sKey, pKey: pub, addr: base58.CheckEncode(hash[1:], hash[0])}
}

func (ledger *Ledger) doInvoke(ch, txID, fn string, args ...string) string {
	resp, err := ledger.doInvokeWithPeerResponse(ch, txID, fn, args...)
	assert.NoError(ledger.t, err)
	assert.Equal(ledger.t, int32(200), resp.Status, resp.Message) //nolint:gomnd
	return string(resp.Payload)
}

func (ledger *Ledger) doInvokeWithErrorReturned(ch, txID, fn string, args ...string) error {
	resp, err := ledger.doInvokeWithPeerResponse(ch, txID, fn, args...)
	if err != nil {
		return err
	}
	if resp.Status != 200 { //nolint:gomnd
		return errors.New(resp.Message)
	}
	return nil
}

func (ledger *Ledger) doInvokeWithPeerResponse(ch, txID, fn string, args ...string) (peer.Response, error) {
	if err := ledger.verifyIncoming(ch, fn); err != nil {
		return peer.Response{}, err
	}
	vArgs := make([][]byte, len(args)+1)
	vArgs[0] = []byte(fn)
	for i, x := range args {
		vArgs[i+1] = []byte(x)
	}

	creator, err := ledger.stubs[ch].GetCreator()
	assert.NoError(ledger.t, err)

	if len(creator) == 0 {
		_ = ledger.stubs[ch].SetDefaultCreatorCert("platformMSP")
	}

	input, err := pb.Marshal(&peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{Name: ch},
			Input:       &peer.ChaincodeInput{Args: vArgs},
		},
	})
	assert.NoError(ledger.t, err)
	payload, err := pb.Marshal(&peer.ChaincodeProposalPayload{Input: input})
	assert.NoError(ledger.t, err)
	proposal, err := pb.Marshal(&peer.Proposal{Payload: payload})
	assert.NoError(ledger.t, err)
	result := ledger.stubs[ch].MockInvokeWithSignedProposal(txID, vArgs, &peer.SignedProposal{
		ProposalBytes: proposal,
	})
	return result, nil
}

// Metadata struct
type Metadata struct {
	Name            string          `json:"name"`
	Symbol          string          `json:"symbol"`
	Decimals        uint            `json:"decimals"`
	UnderlyingAsset string          `json:"underlyingAsset"`
	Issuer          string          `json:"issuer"`
	Methods         []string        `json:"methods"`
	TotalEmission   *big.Int        `json:"total_emission"` //nolint:tagliatelle
	Fee             *Fee            `json:"fee"`
	Rates           []*MetadataRate `json:"rates"`
}

// IndustrialMetadata struct
type IndustrialMetadata struct {
	Name            string          `json:"name"`
	Symbol          string          `json:"symbol"`
	Decimals        uint            `json:"decimals"`
	UnderlyingAsset string          `json:"underlying_asset"` //nolint:tagliatelle
	DeliveryForm    string          `json:"deliveryForm"`
	UnitOfMeasure   string          `json:"unitOfMeasure"`
	TokensForUnit   string          `json:"tokensForUnit"`
	PaymentTerms    string          `json:"paymentTerms"`
	Price           string          `json:"price"`
	Issuer          string          `json:"issuer"`
	Methods         []string        `json:"methods"`
	Groups          []MetadataGroup `json:"groups"`
	Fee             *Fee            `json:"fee"`
	Rates           []*MetadataRate `json:"rates"`
}

// Fee struct
type Fee struct {
	Currency string   `json:"currency"`
	Fee      *big.Int `json:"fee"`
	Floor    *big.Int `json:"floor"`
	Cap      *big.Int `json:"cap"`
}

// MetadataGroup struct
type MetadataGroup struct {
	Name         string    `json:"name"`
	Amount       *big.Int  `json:"amount"`
	MaturityDate time.Time `json:"maturityDate"`
	Note         string    `json:"note"`
}

// MetadataRate struct
type MetadataRate struct {
	DealType string   `json:"deal_type"` //nolint:tagliatelle
	Currency string   `json:"currency"`
	Rate     *big.Int `json:"rate"`
	Min      *big.Int `json:"min"`
	Max      *big.Int `json:"max"`
}

// Metadata returns metadata
func (ledger *Ledger) Metadata(ch string) *Metadata {
	resp := ledger.doInvoke(ch, txIDGen(), "metadata")
	fmt.Println(resp)
	var out Metadata
	err := json.Unmarshal([]byte(resp), &out)
	assert.NoError(ledger.t, err)
	return &out
}

// IndustrialMetadata returns metadata for industrial token
func (ledger *Ledger) IndustrialMetadata(ch string) *IndustrialMetadata {
	resp := ledger.doInvoke(ch, txIDGen(), "metadata")
	fmt.Println(resp)
	var out IndustrialMetadata
	err := json.Unmarshal([]byte(resp), &out)
	assert.NoError(ledger.t, err)

	return &out
}

// MethodExists checks if method exists
func (m Metadata) MethodExists(method string) bool {
	for _, mm := range m.Methods {
		if mm == method {
			return true
		}
	}
	return false
}

func txIDGen() string {
	txID := [16]byte(uuid.New())
	return hex.EncodeToString(txID[:])
}

// GetPending returns pending transactions
func (ledger *Ledger) GetPending(token string, txID ...string) {
	for k, v := range ledger.stubs[token].State {
		if !strings.HasPrefix(k, "\x00"+ledger.batchPrefix+"\x00") {
			continue
		}
		id := strings.Split(k, "\x00")[2]
		if len(txID) == 0 || stringsContains(id, txID) {
			var p proto.PendingTx
			assert.NoError(ledger.t, pb.Unmarshal(v, &p))
			fmt.Println(id, string(p.DumpJSON()))
		}
	}
}

func stringsContains(str string, slice []string) bool {
	for _, s := range slice {
		if str == s {
			return true
		}
	}
	return false
}

func (ledger *Ledger) verifyIncoming(ch string, fn string) error {
	if ch == "" {
		return errors.New("channel undefined")
	}
	if fn == "" {
		return errors.New("chaincode method undefined")
	}
	if _, ok := ledger.stubs[ch]; !ok {
		return fmt.Errorf("stub of [%s] not found", ch)
	}

	return nil
}
