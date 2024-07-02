package unit

import (
	"encoding/json"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/btcsuite/btcutil/base58"
	"github.com/stretchr/testify/require"
)

func TestGroupTxExecutorEmitAndTransfer(t *testing.T) {
	os.Setenv("CORE_CHAINCODE_LOGGING_LEVEL", "DEBUG")
	t.Parallel()

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAggregator := ledger.NewWallet()

	fiat := NewFiatTestToken(token.BaseToken{})
	fiatConfig := makeBaseTokenConfig("fiat", "FIAT", 8,
		owner.Address(), feeSetter.Address(), feeAddressSetter.Address(), "", nil)
	initMsg := ledger.NewCC("fiat", fiat, fiatConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()

	_, err := owner.ExecuteSignedInvoke("fiat", "emit", user1.Address(), "1000")
	require.NoError(t, err)

	user1.BalanceShouldBe("fiat", 1000)

	_, err = feeAddressSetter.ExecuteSignedInvoke("fiat", "setFeeAddress", feeAggregator.Address())
	require.NoError(t, err)
	_, err = feeSetter.ExecuteSignedInvoke("fiat", "setFee", "FIAT", "500000", "100", "100000")
	require.NoError(t, err)

	rawMD := feeSetter.Invoke("fiat", "metadata")
	md := &metadata{}
	require.NoError(t, json.Unmarshal([]byte(rawMD), md))

	require.Equal(t, "FIAT", md.Fee.Currency)
	require.Equal(t, "500000", md.Fee.Fee.String())
	require.Equal(t, "100000", md.Fee.Cap.String())
	require.Equal(t, "100", md.Fee.Floor.String())
	require.Equal(t, feeAggregator.Address(), md.Fee.Address)

	user2 := ledger.NewWallet()
	_, err = user1.ExecuteSignedInvoke("fiat", "transfer", user2.Address(), "400", "")
	require.NoError(t, err)
	user1.BalanceShouldBe("fiat", 500)
	user2.BalanceShouldBe("fiat", 400)

	user1.PublicKeyBase58 = base58.Encode(user1.PublicKeyEd25519)
	_, err = user2.ExecuteSignedInvoke("fiat", "accountsTest", user1.Address(), user1.PublicKeyBase58)
	require.NoError(t, err)
}

func BenchmarkTestGroupTxExecutorEmitAndTransfer(b *testing.B) {
	b.StopTimer()
	err := os.Setenv("CORE_CHAINCODE_LOGGING_LEVEL", "DEBUG")
	if err != nil {
		b.Fatal(err)
	}
	logger.Logger()
	t := &testing.T{}
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAggregator := ledger.NewWallet()

	fiat := NewFiatTestToken(token.BaseToken{})
	fiatConfig := makeBaseTokenConfig("fiat", "FIAT", 8,
		owner.Address(), feeSetter.Address(), feeAddressSetter.Address(), "", nil)
	initMsg := ledger.NewCC("fiat", fiat, fiatConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()

	_, err = owner.ExecuteSignedInvoke("fiat", "emit", user1.Address(), "9999999999999999")
	require.NoError(t, err)

	user1.BalanceShouldBe("fiat", 9999999999999999)

	_, err = feeAddressSetter.ExecuteSignedInvoke("fiat", "setFeeAddress", feeAggregator.Address())
	require.NoError(t, err)
	_, err = feeSetter.ExecuteSignedInvoke("fiat", "setFee", "FIAT", "500000", "100", "100000")
	require.NoError(t, err)

	rawMD := feeSetter.Invoke("fiat", "metadata")
	md := &metadata{}
	require.NoError(t, json.Unmarshal([]byte(rawMD), md))

	require.Equal(t, "FIAT", md.Fee.Currency)
	require.Equal(t, "500000", md.Fee.Fee.String())
	require.Equal(t, "100000", md.Fee.Cap.String())
	require.Equal(t, "100", md.Fee.Floor.String())
	require.Equal(t, feeAggregator.Address(), md.Fee.Address)

	channel := "fiat"
	transferFn := "transfer"
	emitAmount := "1"
	reason := ""
	countInBatch := 1000

	// Artificial delay to update the nonce value.
	time.Sleep(time.Millisecond * 5)

	// Generation of nonce based on current time in milliseconds.
	ms := time.Now().UnixNano() / 1000000

	p := make([]PreparedTask, 0)
	for i := 0; i < b.N; i++ {
		var tasks []*proto.Task
		for i := 0; i < countInBatch; i++ {
			ms++
			nonce := strconv.FormatInt(ms, 10)
			args := []string{ledger.NewWallet().Address(), emitAmount, reason}
			executorRequest := mock.NewExecutorRequest(channel, transferFn, args, true)
			if executorRequest.IsSignedInvoke {
				args = user1.WithNonceSignArgs(executorRequest.Channel, executorRequest.Method, nonce, args...)
			}
			task := &proto.Task{
				Id:     strconv.FormatInt(rand.Int63(), 10),
				Method: executorRequest.Method,
				Args:   args,
			}
			tasks = append(tasks, task)
		}
		p = append(p, PreparedTask{
			tasks: tasks,
		})
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err = user1.TasksExecutor(channel, transferFn, p[i].tasks)
	}
}

type PreparedTask struct {
	tasks []*proto.Task
}
