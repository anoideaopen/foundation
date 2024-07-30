package mock

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcd/btcutil/base58"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
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
	require.NoError(w.ledger.t, err)
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
		require.NoError(w.ledger.t, err)
		return "", TxResponse{}, nil, nil
	}
	txID := txIDGen()
	args, _ = w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	require.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("platformMSP", cert)

	if ctx == nil {
		w.ledger.doInvoke(ch, txID, fn, args...)
	} else {
		w.ledger.doInvokeTraced(ctx, ch, txID, fn, args...)
	}

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

func (l *Ledger) doInvokeTraced(ctx context.Context, ch, txID, fn string, args ...string) string {
	var (
		resp peer.Response
		err  error
	)
	if ctx == nil {
		resp, err = l.doInvokeWithPeerResponse(ch, txID, fn, args...)
	} else {
		resp, err = l.doInvokeWithPeerResponseTraced(ctx, ch, txID, fn, args...)
	}
	require.NoError(l.t, err)
	require.Equal(l.t, int32(200), resp.GetStatus(), resp.GetMessage()) //nolint:gomnd
	return string(resp.GetPayload())
}

// NbInvokeTraced executes non-batched transactions with telemetry tracing
func (w *Wallet) NbInvokeTraced(ctx context.Context, ch string, fn string, args ...string) (string, string) {
	if err := w.verifyIncoming(ch, fn); err != nil {
		require.NoError(w.ledger.t, err)
		return "", ""
	}
	txID := txIDGen()
	message, hash := w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	require.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("platformMSP", cert)

	if ctx == nil {
		w.ledger.doInvoke(ch, txID, fn, message...)
	} else {
		w.ledger.doInvokeTraced(ctx, ch, txID, fn, message...)
	}

	nested, err := pb.Marshal(&proto.Nested{Args: append([]string{fn}, message...)})
	require.NoError(w.ledger.t, err)

	return base58.Encode(nested), hash
}

func (l *Ledger) doInvokeTracedWithErrorReturned(ctx context.Context, ch, txID, fn string, args ...string) error {
	var (
		resp peer.Response
		err  error
	)
	if ctx == nil {
		resp, err = l.doInvokeWithPeerResponse(ch, txID, fn, args...)
	} else {
		resp, err = l.doInvokeWithPeerResponseTraced(ctx, ch, txID, fn, args...)
	}
	if err != nil {
		return err
	}
	if resp.GetStatus() != 200 { //nolint:gomnd
		return errors.New(resp.GetMessage())
	}
	return nil
}

func (l *Ledger) doInvokeWithPeerResponseTraced(ctx context.Context, ch, txID, fn string, args ...string) (peer.Response, error) {
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

	carrier := propagation.MapCarrier{}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	transientDataMap, err := telemetry.PackToTransientMap(carrier)
	require.NoError(l.t, err)

	payload, err := pb.Marshal(&peer.ChaincodeProposalPayload{Input: input, TransientMap: transientDataMap})
	require.NoError(l.t, err)
	proposal, err := pb.Marshal(&peer.Proposal{Payload: payload})
	require.NoError(l.t, err)
	result := l.stubs[ch].MockInvokeWithSignedProposal(txID, vArgs, &peer.SignedProposal{
		ProposalBytes: proposal,
	})
	return result, nil
}
