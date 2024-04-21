package mock

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/anoideaopen/foundation/core"
	"github.com/hyperledger/fabric-protos-go/peer"
)

func (w *Wallet) BatcherSignedInvoke(ch string, fn string, args ...string) (peer.Response, error) {
	err := w.verifyIncoming(ch, fn)
	if err != nil {
		return peer.Response{}, fmt.Errorf("verify incoming args: %w", err)
	}

	// setup creator
	cert, err := hex.DecodeString(batchRobotCert)
	if err != nil {
		return peer.Response{}, fmt.Errorf("decode hex string batchRobotCert: %w", err)
	}
	w.ledger.stubs[ch].SetCreator(cert)

	// sign argument and use output args with signature for invoke chaincode 'batcherBatchExecute'
	argsWithSign, _ := w.sign(fn, ch, args...)

	r := core.BatcherRequest{}
	r.Chaincode = ch
	r.Method = fn
	r.Args = argsWithSign
	r.BatcherRequestType = core.TxBatcherRequestType
	requests := core.BatcherBatchDTO{Requests: []core.BatcherRequest{r}}
	bytes, err := json.Marshal(requests)
	if err != nil {
		return peer.Response{}, fmt.Errorf("marshal requests BatcherBatchDTO: %w", err)
	}

	// do invoke chaincode
	txID := txIDGen()
	peerResponse, err := w.ledger.doInvokeWithPeerResponse(ch, txID, core.BatcherBatchExecute, string(bytes))
	if err != nil {
		return peer.Response{}, fmt.Errorf("doInvokeWithPeerResponseO: %w", err)
	}
	return peerResponse, nil
}
