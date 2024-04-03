package unit

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"
)

// TestAtomicMultiSwapMoveToken moves BA token from ba channel to another channel
func TestAtomicMultiSwapMoveToken(t *testing.T) { //nolint:gocognit
	const (
		tokenBA           = "BA"
		baCC              = "BA"
		otfCC             = "OTF"
		BA1               = "A.101"
		BA2               = "A.102"
		AllowedBalanceBA1 = tokenBA + "_" + BA1
		AllowedBalanceBA2 = tokenBA + "_" + BA2
	)
	m := mock.NewLedger(t)
	issuer := m.NewWallet()
	feeSetter := m.NewWallet()
	feeAddressSetter := m.NewWallet()
	owner := m.NewWallet()
	user1 := m.NewWallet()

	ba := token.BaseToken{
		Name:     baCC,
		Symbol:   baCC,
		Decimals: 8,
	}
	m.NewChainCode(baCC, &ba, &core.ContractOptions{}, nil, issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())

	otf := token.BaseToken{
		Name:     otfCC,
		Symbol:   otfCC,
		Decimals: 8,
	}
	m.NewChainCode(otfCC, &otf, nil, nil, owner.Address())

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
	assert.NoError(t, err)
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
		w.Invoke(ch, "batchExecute", string(data))
		e := <-stub.ChaincodeEventsChannel
		if e.EventName == "batchExecute" {
			events := &proto.BatchEvent{}
			err = pb.Unmarshal(e.Payload, events)
			if err != nil {
				assert.FailNow(t, err.Error())
			}
			for _, ev := range events.Events {
				if hex.EncodeToString(ev.Id) == txID {
					evts := make(map[string][]byte)
					for _, evt := range ev.Events {
						evts[evt.Name] = evt.Value
					}
					if ev.Error != nil {
						assert.FailNow(t, err.Error())
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

	m.WaitMultiSwapAnswer(otfCC, txID, time.Second*5)

	swapID := user1.Invoke(otfCC, "multiSwapGet", txID)
	assert.NotNil(t, swapID)

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
		w.Invoke(ch, "batchExecute", string(data))
		e := <-stub.ChaincodeEventsChannel
		if e.EventName == "batchExecute" {
			events := &proto.BatchEvent{}
			err = pb.Unmarshal(e.Payload, events)
			if err != nil {
				assert.FailNow(t, err.Error())
			}
			for _, ev := range events.Events {
				if hex.EncodeToString(ev.Id) == txID {
					evts := make(map[string][]byte)
					for _, evt := range ev.Events {
						evts[evt.Name] = evt.Value
					}
					if ev.Error != nil {
						assert.FailNow(t, err.Error())
					}
				}
			}
		}
	}
	user1.CheckGivenBalanceShouldBe(baCC, otfCC, 2)
}

// TestAtomicMultiSwapMoveTokenBack moves allowed tokens from external channel to token channel
func TestAtomicMultiSwapMoveTokenBack(t *testing.T) {
	const (
		tokenBA           = "BA"
		baCC              = "BA"
		otfCC             = "OTF"
		BA1               = "A.101"
		BA2               = "A.102"
		AllowedBalanceBA1 = tokenBA + "_" + BA1
		AllowedBalanceBA2 = tokenBA + "_" + BA2
	)

	m := mock.NewLedger(t)
	issuer := m.NewWallet()
	feeSetter := m.NewWallet()
	feeAddressSetter := m.NewWallet()
	owner := m.NewWallet()
	user1 := m.NewWallet()

	ba := token.BaseToken{
		Name:     strings.ToLower(baCC),
		Symbol:   baCC,
		Decimals: 8,
	}
	m.NewChainCode(baCC, &ba, &core.ContractOptions{}, nil, issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())

	otf := token.BaseToken{
		Name:     strings.ToLower(otfCC),
		Symbol:   otfCC,
		Decimals: 8,
	}
	m.NewChainCode(otfCC, &otf, nil, nil, owner.Address())

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
	assert.NoError(t, err)
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

	m.WaitMultiSwapAnswer(baCC, txID, time.Second*5)

	swapID := user1.Invoke(baCC, "multiSwapGet", txID)
	assert.NotNil(t, swapID)

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
	const (
		baCC = "BA"
	)

	m := mock.NewLedger(t)
	issuer := m.NewWallet()
	feeSetter := m.NewWallet()
	feeAddressSetter := m.NewWallet()
	user1 := m.NewWallet()

	ba := token.BaseToken{
		Name:     baCC,
		Symbol:   baCC,
		Decimals: 8,
	}
	m.NewChainCode(baCC, &ba, &core.ContractOptions{DisableMultiSwaps: true}, nil, issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())

	err := user1.RawSignedInvokeWithErrorReturned(baCC, "multiSwapBegin", "", "")
	assert.EqualError(t, err, "method not found: 'multiSwapBegin'")
	err = user1.RawSignedInvokeWithErrorReturned(baCC, "multiSwapCancel", "", "")
	assert.EqualError(t, err, "method not found: 'multiSwapCancel'")
	err = user1.RawSignedInvokeWithErrorReturned(baCC, "multiSwapGet", "", "")
	assert.EqualError(t, err, "method not found: 'multiSwapGet'")
	err = user1.RawSignedInvokeWithErrorReturned(baCC, "multiSwapDone", "", "")
	assert.EqualError(t, err, "multiswaps disabled")
}

// TestAtomicMultiSwapToThirdChannel checks swap/multi swap with third channel is not available
func TestAtomicMultiSwapToThirdChannel(t *testing.T) {
	const (
		tokenBA           = "BA"
		ba02CC            = "BA02"
		otfCC             = "OTF"
		BA1               = "A.101"
		BA2               = "A.102"
		AllowedBalanceBA1 = tokenBA + "_" + BA1
		AllowedBalanceBA2 = tokenBA + "_" + BA2
	)

	m := mock.NewLedger(t)
	owner := m.NewWallet()
	user1 := m.NewWallet()

	otf := token.BaseToken{
		Name:     strings.ToLower(otfCC),
		Symbol:   otfCC,
		Decimals: 8,
	}
	m.NewChainCode(otfCC, &otf, nil, nil, owner.Address())

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
	assert.NoError(t, err)
	_, res, _, _ := user1.RawSignedMultiSwapInvoke(otfCC, "multiSwapBegin", tokenBA, string(bytes), ba02CC, swapHash) //nolint:dogsled
	assert.Equal(t, "incorrect swap", res.Error)
	err = user1.RawSignedInvokeWithErrorReturned(otfCC, "swapBegin", tokenBA, string(bytes), ba02CC, swapHash)
	assert.Error(t, err)
	assert.Equal(t, "incorrect swap", res.Error)
}
