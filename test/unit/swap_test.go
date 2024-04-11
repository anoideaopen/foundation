package unit

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	swapDoneEventCounter      = "__swap_done_event_counter"
	miltiswapDoneEventCounter = "__multiswap_done_event_counter"
)

type FnCallCount struct {
	Count int `json:"count"`
}

type CustomToken struct {
	token.BaseToken
}

// OnSwapDoneEvent is a swap done callback.
func (ct *CustomToken) OnSwapDoneEvent(
	token string,
	owner *types.Address,
	amount *big.Int,
) {
	fmt.Printf(
		"OnSwapEvent(): symbol: %s, token: %s, owner: %s, amount: %s\n",
		ct.ContractConfig().Symbol,
		token,
		owner.String(),
		amount.String(),
	)

	_ = ct.incSwapCallCount()
}

// incSwapCallCount increments OnSwapDoneEvent function call counter.
// Counter stored in chaincode state.
func (ct *CustomToken) incSwapCallCount() error {
	calledBytes, _ := ct.GetStub().GetState(swapDoneEventCounter)
	var fcc FnCallCount
	_ = json.Unmarshal(calledBytes, &fcc)

	fcc.Count++

	calledBytes, _ = json.Marshal(fcc)
	_ = ct.GetStub().PutState(swapDoneEventCounter, calledBytes)

	return nil
}

// QuerySwapDoneEventCallCount fetches OnSwapDoneEvent call counter value.
// Counter stored in chaincode state.
func (ct *CustomToken) QuerySwapDoneEventCallCount() (int, error) {
	calledBytes, _ := ct.GetStub().GetState(swapDoneEventCounter)
	var fcc FnCallCount
	_ = json.Unmarshal(calledBytes, &fcc)

	return fcc.Count, nil
}

func TestAtomicSwap(t *testing.T) {
	t.Parallel()

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	cc := CustomToken{}
	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("cc", &cc, ccConfig)
	require.Empty(t, initMsg)

	vt := CustomToken{}
	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		owner.Address(), "", "", "", nil)
	ledger.NewCC("vt", &vt, vtConfig)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	swapKey := "123"
	hashed := sha3.Sum256([]byte(swapKey))
	swapHash := hex.EncodeToString(hashed[:])

	txID := user1.SignedInvoke("cc", "swapBegin", "CC", "VT", "450", swapHash)
	user1.BalanceShouldBe("cc", 550)
	ledger.WaitSwapAnswer("vt", txID, time.Second*5)

	user1.Invoke("vt", "swapDone", txID, swapKey)
	user1.AllowedBalanceShouldBe("vt", "CC", 450)

	user1.CheckGivenBalanceShouldBe("vt", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 0)

	// TODO: Missed part where robot applies this swapDone in 'cc' channel

	// check swap callback is called exactly 1 time
	fnCountData := user1.Invoke("vt", "swapDoneEventCallCount")
	swapDoneFnCount, err := strconv.Atoi(fnCountData)
	require.NoError(t, err)
	require.Equal(t, 1, swapDoneFnCount)
}

func TestAtomicSwapBack(t *testing.T) {
	t.Parallel()

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	cc := CustomToken{}
	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("cc", &cc, ccConfig)
	require.Empty(t, initMsg)

	vt := CustomToken{}
	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		owner.Address(), "", "", "", nil)
	ledger.NewCC("vt", &vt, vtConfig)

	user1 := ledger.NewWallet()

	user1.AddAllowedBalance("vt", "CC", 1000)
	user1.AddGivenBalance("cc", "VT", 1000)
	user1.AllowedBalanceShouldBe("vt", "CC", 1000)

	swapKey := "123"
	hashed := sha3.Sum256([]byte(swapKey))
	swapHash := hex.EncodeToString(hashed[:])

	txID := user1.SignedInvoke("vt", "swapBegin", "CC", "CC", "450", swapHash)
	ledger.WaitSwapAnswer("cc", txID, time.Second*5)

	user1.Invoke("cc", "swapDone", txID, swapKey)

	user1.AllowedBalanceShouldBe("cc", "CC", 0)
	user1.AllowedBalanceShouldBe("vt", "CC", 550)
	user1.BalanceShouldBe("cc", 450)
	user1.BalanceShouldBe("vt", 0)
	user1.CheckGivenBalanceShouldBe("vt", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 550)
}

func TestAtomicSwapDisableSwaps(t *testing.T) {
	t.Parallel()

	const baCC = "BA"

	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()
	user1 := ledger.NewWallet()

	ba := &token.BaseToken{}

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   baCC,
			Options:  &proto.ChaincodeOptions{DisableSwaps: true},
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &proto.Wallet{Address: fixtures_test.AdminAddr},
		},
		Token: &proto.TokenConfig{
			Name:     "BA Token",
			Decimals: 8,
			Issuer:   &proto.Wallet{Address: issuer.Address()},
		},
	}

	cfgBytes, err := protojson.Marshal(cfg)
	require.NoError(t, err)

	initMsg := ledger.NewCC(baCC, ba, string(cfgBytes))
	require.Empty(t, initMsg)

	err = user1.RawSignedInvokeWithErrorReturned(baCC, "swapBegin", "", "")
	require.ErrorContains(t, err, "method 'swapBegin' not found")
	err = user1.RawSignedInvokeWithErrorReturned(baCC, "swapCancel", "", "")
	require.ErrorContains(t, err, "method 'swapCancel' not found")
	err = user1.RawSignedInvokeWithErrorReturned(baCC, "swapGet", "", "")
	require.ErrorContains(t, err, "method 'swapGet' not found")
	err = user1.RawSignedInvokeWithErrorReturned(baCC, "swapDone", "", "")
	require.ErrorContains(t, err, core.ErrSwapDisabled.Error())
}
