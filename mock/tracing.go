package mock

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func (w *Wallet) SignedInvokeTraced(ctx context.Context, ch, fn string, args ...string) string {
	var (
		txID  string
		res   TxResponse
		swaps []*proto.Swap
	)
	if ctx == nil {
		txID, res, swaps = w.RawSignedInvoke(ch, fn, args...)
	} else {
		txID, res, swaps = w.RawSignedInvokeTraced(ctx, ch, fn, args...)
	}
	assert.Equal(w.ledger.t, "", res.Error)
	for _, swap := range swaps {
		x := proto.Batch{Swaps: []*proto.Swap{{
			Id:      swap.Id,
			Creator: []byte("0000"),
			Owner:   swap.Owner,
			Token:   swap.Token,
			Amount:  swap.Amount,
			From:    swap.From,
			To:      swap.To,
			Hash:    swap.Hash,
			Timeout: swap.Timeout,
		}}}
		data, err := pb.Marshal(&x)
		assert.NoError(w.ledger.t, err)
		cert, err := hex.DecodeString(batchRobotCert)
		assert.NoError(w.ledger.t, err)
		w.ledger.stubs[strings.ToLower(swap.To)].SetCreator(cert)
		w.Invoke(strings.ToLower(swap.To), batchFn, string(data))
	}

	return txID
}

func (w *Wallet) InvokeTraced(ctx context.Context, ch, fn string, args ...string) string {
	if ctx == nil {
		return w.ledger.doInvoke(ch, txIDGen(), fn, args...)
	}
	return w.ledger.doInvokeTraced(ctx, ch, txIDGen(), fn, args...)
}

func (w *Wallet) RawSignedInvokeTracedWithErrorReturned(ctx context.Context, ch, fn string, args ...string) error {
	if err := w.verifyIncoming(ch, fn); err != nil {
		return err
	}
	txID := txIDGen()
	args, _ = w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	assert.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("platformMSP", cert)
	if ctx == nil {
		err = w.ledger.doInvokeWithErrorReturned(ch, txID, fn, args...)
	} else {
		err = w.ledger.doInvokeTracedWithErrorReturned(ctx, ch, txID, fn, args...)
	}
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
	res := w.Invoke(ch, batchFn, string(data))
	out := &proto.BatchResponse{}
	err = pb.Unmarshal([]byte(res), out)
	if err != nil {
		return err
	}

	e := <-w.ledger.stubs[ch].ChaincodeEventsChannel
	if e.EventName == batchFn {
		events := &proto.BatchEvent{}
		err = pb.Unmarshal(e.Payload, events)
		if err != nil {
			return err
		}
		for _, ev := range events.Events {
			if hex.EncodeToString(ev.Id) == txID {
				evts := make(map[string][]byte)
				for _, evt := range ev.Events {
					evts[evt.Name] = evt.Value
				}
				if ev.Error != nil {
					return errors.New(ev.Error.Error)
				}
				return nil
			}
		}
	}
	assert.Fail(w.ledger.t, shouldNotBeHereMsg)
	return nil
}

func (w *Wallet) RawSignedInvokeTraced(ctx context.Context, ch, fn string, args ...string) (string, TxResponse, []*proto.Swap) {
	var (
		invoke   string
		response TxResponse
		swaps    []*proto.Swap
	)
	if ctx == nil {
		invoke, response, swaps, _ = w.RawSignedMultiSwapInvoke(ch, fn, args...)
	} else {
		invoke, response, swaps, _ = w.RawSignedMultiSwapInvokeTraced(ctx, ch, fn, args...)
	}
	return invoke, response, swaps
}

func (w *Wallet) RawSignedMultiSwapInvokeTraced(ctx context.Context, ch, fn string, args ...string) (string, TxResponse, []*proto.Swap, []*proto.MultiSwap) {
	if err := w.verifyIncoming(ch, fn); err != nil {
		assert.NoError(w.ledger.t, err)
		return "", TxResponse{}, nil, nil
	}
	txID := txIDGen()
	args, _ = w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	assert.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("platformMSP", cert)

	if ctx == nil {
		w.ledger.doInvoke(ch, txID, fn, args...)
	} else {
		w.ledger.doInvokeTraced(ctx, ch, txID, fn, args...)
	}

	id, err := hex.DecodeString(txID)
	assert.NoError(w.ledger.t, err)
	data, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{id}})
	assert.NoError(w.ledger.t, err)

	cert, err = hex.DecodeString(batchRobotCert)
	assert.NoError(w.ledger.t, err)
	w.ledger.stubs[ch].SetCreator(cert)
	res := w.Invoke(ch, batchFn, string(data))
	out := &proto.BatchResponse{}
	assert.NoError(w.ledger.t, pb.Unmarshal([]byte(res), out))

	e := <-w.ledger.stubs[ch].ChaincodeEventsChannel
	if e.EventName == batchFn {
		events := &proto.BatchEvent{}
		assert.NoError(w.ledger.t, pb.Unmarshal(e.Payload, events))
		for _, ev := range events.Events {
			if hex.EncodeToString(ev.Id) == txID {
				evts := make(map[string][]byte)
				for _, evt := range ev.Events {
					evts[evt.Name] = evt.Value
				}
				er := ""
				if ev.Error != nil {
					er = ev.Error.Error
				}
				return txID, TxResponse{
					Method: ev.Method,
					Error:  er,
					Result: string(ev.Result),
					Events: evts,
				}, out.CreatedSwaps, out.CreatedMultiSwap
			}
		}
	}
	assert.Fail(w.ledger.t, shouldNotBeHereMsg)
	return txID, TxResponse{}, out.CreatedSwaps, out.CreatedMultiSwap
}

func (ledger *Ledger) doInvokeTraced(ctx context.Context, ch, txID, fn string, args ...string) string {
	var (
		resp peer.Response
		err  error
	)
	if ctx == nil {
		resp, err = ledger.doInvokeWithPeerResponse(ch, txID, fn, args...)
	} else {
		resp, err = ledger.doInvokeWithPeerResponseTraced(ctx, ch, txID, fn, args...)
	}
	assert.NoError(ledger.t, err)
	assert.Equal(ledger.t, int32(200), resp.Status, resp.Message) //nolint:gomnd
	return string(resp.Payload)
}

// NbInvokeTraced executes non-batched transactions with telemetry tracing
func (w *Wallet) NbInvokeTraced(ctx context.Context, ch string, fn string, args ...string) (string, string) {
	if err := w.verifyIncoming(ch, fn); err != nil {
		assert.NoError(w.ledger.t, err)
		return "", ""
	}
	txID := txIDGen()
	message, hash := w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	assert.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("platformMSP", cert)

	if ctx == nil {
		w.ledger.doInvoke(ch, txID, fn, message...)
	} else {
		w.ledger.doInvokeTraced(ctx, ch, txID, fn, message...)
	}

	nested, err := pb.Marshal(&proto.Nested{Args: append([]string{fn}, message...)})
	assert.NoError(w.ledger.t, err)

	return base58.Encode(nested), hash
}

func (ledger *Ledger) doInvokeTracedWithErrorReturned(ctx context.Context, ch, txID, fn string, args ...string) error {
	var (
		resp peer.Response
		err  error
	)
	if ctx == nil {
		resp, err = ledger.doInvokeWithPeerResponse(ch, txID, fn, args...)
	} else {
		resp, err = ledger.doInvokeWithPeerResponseTraced(ctx, ch, txID, fn, args...)
	}
	if err != nil {
		return err
	}
	if resp.Status != 200 { //nolint:gomnd
		return errors.New(resp.Message)
	}
	return nil
}

func (ledger *Ledger) doInvokeWithPeerResponseTraced(ctx context.Context, ch, txID, fn string, args ...string) (peer.Response, error) {
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

	carrier := propagation.MapCarrier{}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	transientDataMap, err := telemetry.PackToTransientMap(carrier)
	assert.NoError(ledger.t, err)

	payload, err := pb.Marshal(&peer.ChaincodeProposalPayload{Input: input, TransientMap: transientDataMap})
	assert.NoError(ledger.t, err)
	proposal, err := pb.Marshal(&peer.Proposal{Payload: payload})
	assert.NoError(ledger.t, err)
	result := ledger.stubs[ch].MockInvokeWithSignedProposal(txID, vArgs, &peer.SignedProposal{
		ProposalBytes: proposal,
	})
	return result, nil
}
