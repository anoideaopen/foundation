package unit

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/anoideaopen/foundation/token"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/encoding/protojson"
)

// OnMultiSwapDoneEvent is a multi-swap done callback.
func (ct *CustomToken) OnMultiSwapDoneEvent(
	token string,
	owner *types.Address,
	assets []*proto.Asset,
) {
	type asset struct {
		group  string
		amount string
	}

	var al []asset
	for _, a := range assets {
		amount := new(big.Int)
		amount.SetBytes(a.Amount)
		al = append(al, asset{
			group:  a.Group,
			amount: amount.String(),
		})
	}

	fmt.Printf(
		"OnMultiSwapDoneEvent(): symbol: %s, token: %s, owner: %s, assets: %v\n",
		ct.ContractConfig().Symbol,
		token,
		owner.String(),
		al,
	)

	_ = ct.incrMultiSwapCallCount()
}

// incrMultiSwapCallCount increments OnMultiSwapDoneEvent function call counter.
// Counter stored in chaincode state.
func (ct *CustomToken) incrMultiSwapCallCount() error {
	calledBytes, _ := ct.GetStub().GetState(miltiswapDoneEventCounter)
	var fcc FnCallCount
	_ = json.Unmarshal(calledBytes, &fcc)

	fcc.Count++

	calledBytes, _ = json.Marshal(fcc)
	_ = ct.GetStub().PutState(miltiswapDoneEventCounter, calledBytes)

	return nil
}

// QueryMultiSwapDoneEventCallCount fetches OnMultiSwapDoneEvent call counter value.
// Counter stored in chaincode state.
func (ct *CustomToken) QueryMultiSwapDoneEventCallCount() (int, error) {
	calledBytes, _ := ct.GetStub().GetState(miltiswapDoneEventCounter)
	var fcc FnCallCount
	_ = json.Unmarshal(calledBytes, &fcc)

	return fcc.Count, nil
}

// TestAtomicMultiSwapMoveToken moves BA token from ba channel to another channel
func TestAtomicMultiSwapMoveToken(t *testing.T) { //nolint:gocognit
	t.Parallel()

	const (
		tokenBA           = "BA"
		baCC              = "BA"
		otfCC             = "OTF"
		BA1               = "A.101"
		BA2               = "A.102"
		AllowedBalanceBA1 = tokenBA + "_" + BA1
		AllowedBalanceBA2 = tokenBA + "_" + BA2
	)
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()
	owner := ledger.NewWallet()
	user1 := ledger.NewWallet()

	ba := &token.BaseToken{}
	baConfig := makeBaseTokenConfig("BA Token", baCC, 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC(baCC, ba, baConfig)
	require.Empty(t, initMsg)

	otf := &CustomToken{}
	otfConfig := makeBaseTokenConfig("OTF Token", otfCC, 8,
		owner.Address(), "", "", "", nil)
	initMsg = ledger.NewCC(otfCC, otf, otfConfig)
	require.Empty(t, initMsg)

	user1.AddTokenBalance(baCC, BA1, 1)
	user1.AddTokenBalance(baCC, BA2, 1)

	user1.GroupBalanceShouldBe(baCC, BA1, 1)
	user1.GroupBalanceShouldBe(baCC, BA2, 1)
	user1.GroupBalanceShouldBe(otfCC, BA1, 0)
	user1.GroupBalanceShouldBe(otfCC, BA2, 0)

	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA2, 0)

	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA1, 0)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA2, 0)

	user1.AllowedBalanceShouldBe(baCC, BA1, 0)
	user1.AllowedBalanceShouldBe(baCC, BA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA1, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA2, 0)

	swapKey := "123"
	hashed := sha3.Sum256([]byte(swapKey))
	swapHash := hex.EncodeToString(hashed[:])

	bytes, err := json.Marshal(types.MultiSwapAssets{
		Assets: []*types.MultiSwapAsset{
			{
				Group:  AllowedBalanceBA1,
				Amount: "1",
			},
			{
				Group:  AllowedBalanceBA2,
				Amount: "1",
			},
		},
	})
	require.NoError(t, err)
	txID, _, _, multiSwaps := user1.RawSignedMultiSwapInvoke(baCC, "multiSwapBegin", tokenBA, string(bytes), otfCC, swapHash)
	w := user1
	for _, swap := range multiSwaps {
		x := proto.Batch{
			MultiSwaps: []*proto.MultiSwap{
				{
					Id:      swap.Id,
					Creator: []byte("0000"),
					Owner:   swap.Owner,
					Token:   swap.Token,
					Assets:  swap.Assets,
					From:    swap.From,
					To:      swap.To,
					Hash:    swap.Hash,
					Timeout: swap.Timeout,
				},
			},
		}
		data, _ := pb.Marshal(&x)
		cert, _ := hex.DecodeString(BatchRobotCert)
		ch := swap.To
		stub := w.Ledger().GetStub(ch)
		stub.SetCreator(cert)
		w.Invoke(ch, core.BatchExecute, string(data))
		e := <-stub.ChaincodeEventsChannel
		if e.EventName == core.BatchExecute {
			events := &proto.BatchEvent{}
			err = pb.Unmarshal(e.Payload, events)
			if err != nil {
				require.FailNow(t, err.Error())
			}
			for _, ev := range events.Events {
				if hex.EncodeToString(ev.Id) == txID {
					evts := make(map[string][]byte)
					for _, evt := range ev.Events {
						evts[evt.Name] = evt.Value
					}
					if ev.Error != nil {
						require.FailNow(t, ev.GetError().GetError())
					}
				}
			}
		}
	}

	user1.GroupBalanceShouldBe(baCC, BA1, 0)
	user1.GroupBalanceShouldBe(baCC, BA2, 0)
	user1.GroupBalanceShouldBe(otfCC, BA1, 0)
	user1.GroupBalanceShouldBe(otfCC, BA2, 0)

	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA2, 0)

	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA1, 0)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA2, 0)

	user1.AllowedBalanceShouldBe(baCC, BA1, 0)
	user1.AllowedBalanceShouldBe(baCC, BA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA1, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA2, 0)

	ledger.WaitMultiSwapAnswer(otfCC, txID, time.Second*5)

	swapID := user1.Invoke(otfCC, "multiSwapGet", txID)
	require.NotNil(t, swapID)

	user1.Invoke(otfCC, "multiSwapDone", txID, swapKey)

	user1.GroupBalanceShouldBe(baCC, BA1, 0)
	user1.GroupBalanceShouldBe(baCC, BA2, 0)
	user1.GroupBalanceShouldBe(otfCC, BA1, 0)
	user1.GroupBalanceShouldBe(otfCC, BA2, 0)

	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA2, 0)

	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA1, 1)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA2, 1)

	user1.AllowedBalanceShouldBe(baCC, BA1, 0)
	user1.AllowedBalanceShouldBe(baCC, BA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA1, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA2, 0)

	// update GivenBalance using batchExecute with MultiSwapsKeys
	for _, swap := range multiSwaps {
		x := proto.Batch{
			MultiSwapsKeys: []*proto.SwapKey{
				{
					Id:  swap.Id,
					Key: swapKey,
				},
			},
		}
		data, _ := pb.Marshal(&x)
		cert, _ := hex.DecodeString(BatchRobotCert)
		ch := swap.From
		stub := w.Ledger().GetStub(ch)
		stub.SetCreator(cert)
		w.Invoke(ch, core.BatchExecute, string(data))
		e := <-stub.ChaincodeEventsChannel
		if e.EventName == core.BatchExecute {
			events := &proto.BatchEvent{}
			err = pb.Unmarshal(e.Payload, events)
			if err != nil {
				require.FailNow(t, err.Error())
			}
			for _, ev := range events.Events {
				if hex.EncodeToString(ev.Id) == txID {
					evts := make(map[string][]byte)
					for _, evt := range ev.Events {
						evts[evt.Name] = evt.Value
					}
					if ev.Error != nil {
						require.FailNow(t, ev.GetError().GetError())
					}
				}
			}
		}
	}
	user1.CheckGivenBalanceShouldBe(baCC, otfCC, 2)

	// check MultiSwap callback is called exactly 1 time
	fnCountData := user1.Invoke(otfCC, "multiSwapDoneEventCallCount")
	swapDoneFnCount, err := strconv.Atoi(fnCountData)
	require.NoError(t, err)
	require.Equal(t, 1, swapDoneFnCount)
}

// TestAtomicMultiSwapMoveTokenBack moves allowed tokens from external channel to token channel
func TestAtomicMultiSwapMoveTokenBack(t *testing.T) {
	t.Parallel()

	const (
		tokenBA           = "BA"
		baCC              = "BA"
		otfCC             = "OTF"
		BA1               = "A.101"
		BA2               = "A.102"
		AllowedBalanceBA1 = tokenBA + "_" + BA1
		AllowedBalanceBA2 = tokenBA + "_" + BA2
	)

	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()
	owner := ledger.NewWallet()
	user1 := ledger.NewWallet()

	ba := &token.BaseToken{}
	baConfig := makeBaseTokenConfig("BA Token", baCC, 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC(baCC, ba, baConfig)
	require.Empty(t, initMsg)

	otf := &CustomToken{}
	otfConfig := makeBaseTokenConfig("OTF Token", otfCC, 8,
		owner.Address(), "", "", "", nil)
	initMsg = ledger.NewCC(otfCC, otf, otfConfig)
	require.Empty(t, initMsg)

	user1.AddGivenBalance(baCC, otfCC, 2)
	user1.CheckGivenBalanceShouldBe(baCC, otfCC, 2)

	user1.AddAllowedBalance(otfCC, AllowedBalanceBA1, 1)
	user1.AddAllowedBalance(otfCC, AllowedBalanceBA2, 1)

	user1.GroupBalanceShouldBe(baCC, BA1, 0)
	user1.GroupBalanceShouldBe(baCC, BA2, 0)
	user1.GroupBalanceShouldBe(otfCC, BA1, 0)
	user1.GroupBalanceShouldBe(otfCC, BA2, 0)

	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA2, 0)

	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA1, 1)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA2, 1)

	user1.AllowedBalanceShouldBe(baCC, BA1, 0)
	user1.AllowedBalanceShouldBe(baCC, BA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA1, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA2, 0)

	swapKey := "123"
	hashed := sha3.Sum256([]byte(swapKey))
	swapHash := hex.EncodeToString(hashed[:])

	bytes, err := json.Marshal(types.MultiSwapAssets{
		Assets: []*types.MultiSwapAsset{
			{
				Group:  AllowedBalanceBA1,
				Amount: "1",
			},
			{
				Group:  AllowedBalanceBA2,
				Amount: "1",
			},
		},
	})
	require.NoError(t, err)
	txID := user1.SignedMultiSwapsInvoke(otfCC, "multiSwapBegin", tokenBA, string(bytes), baCC, swapHash)

	user1.GroupBalanceShouldBe(baCC, BA1, 0)
	user1.GroupBalanceShouldBe(baCC, BA2, 0)
	user1.GroupBalanceShouldBe(otfCC, BA1, 0)
	user1.GroupBalanceShouldBe(otfCC, BA2, 0)

	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA2, 0)

	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA1, 0)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA2, 0)

	user1.AllowedBalanceShouldBe(baCC, BA1, 0)
	user1.AllowedBalanceShouldBe(baCC, BA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA1, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA2, 0)

	ledger.WaitMultiSwapAnswer(baCC, txID, time.Second*5)

	swapID := user1.Invoke(baCC, "multiSwapGet", txID)
	require.NotNil(t, swapID)

	user1.CheckGivenBalanceShouldBe(baCC, otfCC, 0)
	user1.GroupBalanceShouldBe(baCC, BA1, 0)
	user1.GroupBalanceShouldBe(baCC, BA2, 0)

	user1.Invoke(baCC, "multiSwapDone", txID, swapKey)

	user1.CheckGivenBalanceShouldBe(baCC, otfCC, 0)

	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.AllowedBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA1, 0)
	user1.AllowedBalanceShouldBe(otfCC, AllowedBalanceBA2, 0)

	user1.AllowedBalanceShouldBe(baCC, BA1, 0)
	user1.AllowedBalanceShouldBe(baCC, BA2, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA1, 0)
	user1.AllowedBalanceShouldBe(otfCC, BA2, 0)

	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(baCC, AllowedBalanceBA2, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA1, 0)
	user1.GroupBalanceShouldBe(otfCC, AllowedBalanceBA2, 0)

	user1.GroupBalanceShouldBe(baCC, BA1, 1)
	user1.GroupBalanceShouldBe(baCC, BA2, 1)
	user1.GroupBalanceShouldBe(otfCC, BA1, 0)
	user1.GroupBalanceShouldBe(otfCC, BA2, 0)
}

func TestAtomicMultiSwapDisableMultiSwaps(t *testing.T) {
	t.Parallel()

	const baCC = "BA"

	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()
	user1 := ledger.NewWallet()

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   baCC,
			Options:  &proto.ChaincodeOptions{DisableMultiSwaps: true},
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

	initMsg := ledger.NewCC(baCC, &token.BaseToken{}, string(cfgBytes))
	require.Empty(t, initMsg)

	err = user1.RawSignedInvokeWithErrorReturned(baCC, "multiSwapBegin", "", "")
	require.ErrorContains(t, err, "method 'multiSwapBegin' not found")
	err = user1.RawSignedInvokeWithErrorReturned(baCC, "multiSwapCancel", "", "")
	require.ErrorContains(t, err, "method 'multiSwapCancel' not found")
	err = user1.RawSignedInvokeWithErrorReturned(baCC, "multiSwapGet", "", "")
	require.ErrorContains(t, err, "method 'multiSwapGet' not found")
	err = user1.RawSignedInvokeWithErrorReturned(baCC, "multiSwapDone", "", "")
	require.ErrorContains(t, err, core.ErrMultiSwapDisabled.Error())
}

// TestAtomicMultiSwapToThirdChannel checks swap/multi swap with third channel is not available
func TestAtomicMultiSwapToThirdChannel(t *testing.T) {
	t.Parallel()

	const (
		tokenBA           = "BA"
		ba02CC            = "BA02"
		otfCC             = "OTF"
		BA1               = "A.101"
		BA2               = "A.102"
		AllowedBalanceBA1 = tokenBA + "_" + BA1
		AllowedBalanceBA2 = tokenBA + "_" + BA2
	)

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()
	user1 := ledger.NewWallet()

	otf := &CustomToken{}
	otfConfig := makeBaseTokenConfig(strings.ToLower(otfCC), otfCC, 8,
		owner.Address(), "", "", "", nil)
	ledger.NewCC(otfCC, otf, otfConfig)

	swapKey := "123"
	hashed := sha3.Sum256([]byte(swapKey))
	swapHash := hex.EncodeToString(hashed[:])

	bytes, err := json.Marshal(types.MultiSwapAssets{
		Assets: []*types.MultiSwapAsset{
			{
				Group:  AllowedBalanceBA1,
				Amount: "1",
			},
			{
				Group:  AllowedBalanceBA2,
				Amount: "1",
			},
		},
	})
	require.NoError(t, err)
	_, res, _, _ := user1.RawSignedMultiSwapInvoke(otfCC, "multiSwapBegin", tokenBA, string(bytes), ba02CC, swapHash) //nolint:dogsled
	require.Equal(t, "incorrect swap", res.Error)
	err = user1.RawSignedInvokeWithErrorReturned(otfCC, "swapBegin", tokenBA, string(bytes), ba02CC, swapHash)
	require.Error(t, err)
	require.Equal(t, "incorrect swap", res.Error)
}
