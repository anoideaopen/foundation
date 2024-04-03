package mock

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/base64"
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
	"github.com/anoideaopen/foundation/mock/stub"
	"github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

const defaultCert = `MIICSjCCAfGgAwIBAgIRAKeZTS2c/qkXBN0Vkh+0WYQwCgYIKoZIzj0EAwIwgYcx
CzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4g
RnJhbmNpc2NvMSMwIQYDVQQKExphdG9teXplLnVhdC5kbHQuYXRvbXl6ZS5jaDEm
MCQGA1UEAxMdY2EuYXRvbXl6ZS51YXQuZGx0LmF0b215emUuY2gwHhcNMjAxMDEz
MDg1NjAwWhcNMzAxMDExMDg1NjAwWjB3MQswCQYDVQQGEwJVUzETMBEGA1UECBMK
Q2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZyYW5jaXNjbzEPMA0GA1UECxMGY2xp
ZW50MSowKAYDVQQDDCFVc2VyMTBAYXRvbXl6ZS51YXQuZGx0LmF0b215emUuY2gw
WTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAR3V6z/nVq66HBDxFFN3/3rUaJLvHgW
FzoKaA/qZQyV919gdKr82LDy8N2kAYpAcP7dMyxMmmGOPbo53locYWIyo00wSzAO
BgNVHQ8BAf8EBAMCB4AwDAYDVR0TAQH/BAIwADArBgNVHSMEJDAigCBSv0ueZaB3
qWu/AwOtbOjaLd68woAqAklfKKhfu10K+DAKBggqhkjOPQQDAgNHADBEAiBFB6RK
O7huI84Dy3fXeA324ezuqpJJkfQOJWkbHjL+pQIgFKIqBJrDl37uXNd3eRGJTL+o
21ZL8pGXH8h0nHjOF9M=`

const adminCert = `MIICSDCCAe6gAwIBAgIQAJwYy5PJAYSC1i0UgVN5bjAKBggqhkjOPQQDAjCBhzEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xIzAhBgNVBAoTGmF0b215emUudWF0LmRsdC5hdG9teXplLmNoMSYw
JAYDVQQDEx1jYS5hdG9teXplLnVhdC5kbHQuYXRvbXl6ZS5jaDAeFw0yMDEwMTMw
ODU2MDBaFw0zMDEwMTEwODU2MDBaMHUxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMQ4wDAYDVQQLEwVhZG1p
bjEpMCcGA1UEAwwgQWRtaW5AYXRvbXl6ZS51YXQuZGx0LmF0b215emUuY2gwWTAT
BgcqhkjOPQIBBggqhkjOPQMBBwNCAAQGQX9IhgjCtd3mYZ9DUszmUgvubepVMPD5
FlwjCglB2SiWuE2rT/T5tHJsU/Y9ZXFtOOpy/g9tQ/0wxDWwpkbro00wSzAOBgNV
HQ8BAf8EBAMCB4AwDAYDVR0TAQH/BAIwADArBgNVHSMEJDAigCBSv0ueZaB3qWu/
AwOtbOjaLd68woAqAklfKKhfu10K+DAKBggqhkjOPQQDAgNIADBFAiEAoKRQLe4U
FfAAwQs3RCWpevOPq+J8T4KEsYvswKjzfJYCIAs2kOmN/AsVUF63unXJY0k9ktfD
fAaqNRaboY1Yg1iQ`

// Ledger is a mock ledger for testing purposes
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
	time.Local = time.UTC
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

	prefix := "batchTransactions"
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

const batchRobotCertHash = "380499dcb3d3ee374ccfd74cbdcbe03a1cd5ae66b282e5673dcb13cbe290965b" //nolint:gosec

// NewChainCode creates new chaincode
func (ledger *Ledger) NewChainCode(name string, bci core.BaseContractInterface, options *core.ContractOptions, fs *embed.FS, initArgs ...string) string {
	_, exists := ledger.stubs[name]
	assert.False(ledger.t, exists)
	cc, err := core.NewCC(bci, options, core.WithSrcFS(fs))
	assert.NoError(ledger.t, err)
	ledger.stubs[name] = stub.NewMockStub(name, cc)
	ledger.stubs[name].ChannelID = name
	ledger.stubs[name].MockPeerChaincode("acl/acl", ledger.stubs["acl"])
	args := [][]byte{[]byte(""), []byte(batchRobotCertHash)}
	for _, arg := range initArgs {
		args = append(args, []byte(arg))
	}
	cert, err := base64.StdEncoding.DecodeString(adminCert)
	assert.NoError(ledger.t, err)
	_ = ledger.stubs[name].SetCreatorCert("atomyzeMSP", cert)
	res := ledger.stubs[name].MockInit(txIDGen(), args)
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
	hash := sha3.Sum256(pKey)
	return &Wallet{ledger: ledger, sKey: sKey, pKey: pKey, addr: base58.CheckEncode(hash[1:], hash[0])}
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

func (ledger *Ledger) doInvoke(ch string, txID string, fn string, args ...string) string {
	if err := ledger.verifyIncoming(ch, fn); err != nil {
		assert.NoError(ledger.t, err)
		return ""
	}
	vArgs := make([][]byte, len(args)+1)
	vArgs[0] = []byte(fn)
	for i, x := range args {
		vArgs[i+1] = []byte(x)
	}

	creator, err := ledger.stubs[ch].GetCreator()
	assert.NoError(ledger.t, err)

	if len(creator) == 0 {
		cert, err := base64.StdEncoding.DecodeString(defaultCert)
		assert.NoError(ledger.t, err)
		_ = ledger.stubs[ch].SetCreatorCert("atomyzeMSP", cert)
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
	assert.Equal(ledger.t, int32(200), result.Status, result.Message) //nolint:gomnd
	return string(result.Payload)
}

func (ledger *Ledger) doInvokeWithErrorReturned(ch string, txID string, fn string, args ...string) error {
	if err := ledger.verifyIncoming(ch, fn); err != nil {
		return err
	}
	vArgs := make([][]byte, len(args)+1)
	vArgs[0] = []byte(fn)
	for i, x := range args {
		vArgs[i+1] = []byte(x)
	}

	creator, err := ledger.stubs[ch].GetCreator()
	assert.NoError(ledger.t, err)

	if len(creator) == 0 {
		cert, err := base64.StdEncoding.DecodeString(defaultCert)
		assert.NoError(ledger.t, err)
		_ = ledger.stubs[ch].SetCreatorCert("atomyzeMSP", cert)
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
	if result.Status != 200 { //nolint:gomnd
		return errors.New(result.Message)
	}
	return nil
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
