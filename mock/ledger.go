package mock

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/cctransfer"
	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/multiswap"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock/stub"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// Deprecated: use package ../mocks instead
type Ledger struct {
	t                   *testing.T
	stubs               map[string]*stub.Stub
	keyEvents           map[string]chan *peer.ChaincodeEvent
	txResponseEvents    map[string]chan TxResponse
	txResponseEventLock *sync.Mutex
	batchPrefix         string
}

// Deprecated: use package ../mocks instead
// GetStubByKey returns stub by key
func (l *Ledger) GetStubByKey(key string) *stub.Stub {
	return l.stubs[key]
}

// Deprecated: use package ../mocks instead
// UpdateStubTxID updates stub txID
func (l *Ledger) UpdateStubTxID(stubName string, newTxID string) {
	l.stubs[stubName].TxID = newTxID
}

// Deprecated: use package ../mocks instead
// NewLedger creates new ledger
func NewLedger(t *testing.T, options ...string) *Ledger {
	lvl := logrus.ErrorLevel
	var err error
	if level, ok := os.LookupEnv("LOG"); ok {
		lvl, err = logrus.ParseLevel(level)
		require.NoError(t, err)
	}
	logrus.SetLevel(lvl)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	aclStub := stub.NewMockStub("acl", new(mockACL))
	resp := aclStub.MockInit(hex.EncodeToString([]byte("acl")), nil)
	require.Equal(t, int32(http.StatusOK), resp.GetStatus())

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

// Deprecated: use package ../mocks instead
// SetACL sets acl stub
func (l *Ledger) SetACL(aclStub *stub.Stub) {
	l.stubs["acl"] = aclStub
}

// TxResponse returns txResponse event
type TxResponse struct {
	Method     string                    `json:"method"`
	Error      string                    `json:"error,omitempty"`
	Result     string                    `json:"result"`
	Events     map[string][]byte         `json:"events,omitempty"`
	Accounting []*proto.AccountingRecord `json:"accounting"`
}

// Deprecated: use package ../mocks instead
func (l *Ledger) NewCC(
	name string,
	bci core.BaseContractInterface,
	config string,
	opts ...core.ChaincodeOption,
) string {
	_, exists := l.stubs[name]
	require.False(
		l.t,
		exists,
		fmt.Sprintf("stub with name '%s' has already exist in ledger mock; "+
			"try to use other chaincode name.", name),
	)

	cc, err := core.NewCC(bci, opts...)
	require.NoError(l.t, err)
	l.stubs[name] = stub.NewMockStub(name, cc)
	l.stubs[name].ChannelID = name

	l.stubs[name].MockPeerChaincode("acl/acl", l.stubs["acl"])

	err = l.stubs[name].SetAdminCreatorCert("platformMSP")
	require.NoError(l.t, err)
	res := l.stubs[name].MockInit(txIDGen(), [][]byte{[]byte(config)})
	message := res.GetMessage()
	if message != "" {
		return message
	}

	l.keyEvents[name] = make(chan *peer.ChaincodeEvent, 1)
	return ""
}

// Deprecated: use package ../mocks instead
// GetStub returns stub
func (l *Ledger) GetStub(name string) *stub.Stub {
	return l.stubs[name]
}

// WaitMultiSwapAnswer waits for multi swap answer
func (l *Ledger) WaitMultiSwapAnswer(name string, id string, timeout time.Duration) {
	interval := time.Second / 2 //nolint:gomnd
	ticker := time.NewTicker(interval)
	count := timeout.Microseconds() / interval.Microseconds()
	key, err := l.stubs[name].CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{id}) //nolint:staticcheck
	require.NoError(l.t, err)
	for count > 0 {
		count--
		<-ticker.C
		if _, exists := l.stubs[name].State[key]; exists {
			return
		}
	}
	for k, v := range l.stubs[name].State {
		fmt.Println(k, string(v))
	}
	require.Fail(l.t, "timeout exceeded")
}

// Deprecated: use package ../mocks instead
// WaitSwapAnswer waits for swap answer
func (l *Ledger) WaitSwapAnswer(name string, id string, timeout time.Duration) {
	interval := time.Second / 2 //nolint:gomnd
	ticker := time.NewTicker(interval)
	count := timeout.Microseconds() / interval.Microseconds()
	key, err := l.stubs[name].CreateCompositeKey("swaps", []string{id})
	require.NoError(l.t, err)
	for count > 0 {
		count--
		<-ticker.C
		if _, exists := l.stubs[name].State[key]; exists {
			return
		}
	}
	for k, v := range l.stubs[name].State {
		fmt.Println(k, string(v))
	}
	require.Fail(l.t, "timeout exceeded")
}

// Deprecated: use package ../mocks instead
// WaitChTransferTo waits for transfer to event
func (l *Ledger) WaitChTransferTo(name string, id string, timeout time.Duration) {
	interval := time.Second / 2 //nolint:gomnd
	ticker := time.NewTicker(interval)
	count := timeout.Microseconds() / interval.Microseconds()
	key := cctransfer.CCToTransfer(id)
	for count > 0 {
		count--
		<-ticker.C
		if _, exists := l.stubs[name].State[key]; exists {
			return
		}
	}
	for k, v := range l.stubs[name].State {
		fmt.Println(k, string(v))
	}
	require.Fail(l.t, "timeout exceeded")
}

// Deprecated: use package ../mocks instead
func (l *Ledger) doInvoke(ch, txID, fn string, args ...string) string {
	resp, err := l.doInvokeWithPeerResponse(ch, txID, fn, args...)
	require.NoError(l.t, err)
	require.Equal(l.t, int32(200), resp.GetStatus(), resp.GetMessage()) //nolint:gomnd
	return string(resp.GetPayload())
}

// Deprecated: use package ../mocks instead
func (l *Ledger) doInvokeWithErrorReturned(ch, txID, fn string, args ...string) error {
	resp, err := l.doInvokeWithPeerResponse(ch, txID, fn, args...)
	if err != nil {
		return err
	}
	if resp.GetStatus() != 200 { //nolint:gomnd
		return errors.New(resp.GetMessage())
	}
	return nil
}

// Deprecated: use package ../mocks instead
func (l *Ledger) doInvokeWithPeerResponse(ch, txID, fn string, args ...string) (peer.Response, error) {
	if err := l.verifyIncoming(ch, fn); err != nil {
		return peer.Response{}, err
	}
	vArgs := make([][]byte, len(args)+1)
	vArgs[0] = []byte(fn)
	for i, x := range args {
		vArgs[i+1] = []byte(x)
	}

	creator, err := l.stubs[ch].GetCreator()
	require.NoError(l.t, err)

	if len(creator) == 0 {
		_ = l.stubs[ch].SetDefaultCreatorCert("platformMSP")
	}

	input, err := pb.Marshal(&peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{Name: ch},
			Input:       &peer.ChaincodeInput{Args: vArgs},
		},
	})
	require.NoError(l.t, err)
	payload, err := pb.Marshal(&peer.ChaincodeProposalPayload{Input: input})
	require.NoError(l.t, err)
	proposal, err := pb.Marshal(&peer.Proposal{Payload: payload})
	require.NoError(l.t, err)
	result := l.stubs[ch].MockInvokeWithSignedProposal(txID, vArgs, &peer.SignedProposal{
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

// Deprecated: use package ../mocks instead
// Metadata returns metadata
func (l *Ledger) Metadata(ch string) *Metadata {
	resp := l.doInvoke(ch, txIDGen(), "metadata")
	fmt.Println(resp)
	var out Metadata
	err := json.Unmarshal([]byte(resp), &out)
	require.NoError(l.t, err)
	return &out
}

// Deprecated: use package ../mocks instead
// IndustrialMetadata returns metadata for industrial token
func (l *Ledger) IndustrialMetadata(ch string) *IndustrialMetadata {
	resp := l.doInvoke(ch, txIDGen(), "metadata")
	fmt.Println(resp)
	var out IndustrialMetadata
	err := json.Unmarshal([]byte(resp), &out)
	require.NoError(l.t, err)

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

// Deprecated: use package ../mocks instead
// GetPending returns pending transactions
func (l *Ledger) GetPending(token string, txID ...string) {
	for k, v := range l.stubs[token].State {
		if !strings.HasPrefix(k, "\x00"+l.batchPrefix+"\x00") {
			continue
		}
		id := strings.Split(k, "\x00")[2]
		if len(txID) == 0 || stringsContains(id, txID) {
			var p proto.PendingTx
			require.NoError(l.t, pb.Unmarshal(v, &p))
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

func (l *Ledger) verifyIncoming(ch string, fn string) error {
	if ch == "" {
		return errors.New("channel undefined")
	}
	if fn == "" {
		return errors.New("chaincode method undefined")
	}
	if _, ok := l.stubs[ch]; !ok {
		return fmt.Errorf("stub of [%s] not found", ch)
	}

	return nil
}
