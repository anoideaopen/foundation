package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"

	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client/types"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	ab "github.com/hyperledger/fabric-protos-go/orderer"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	"github.com/hyperledger/fabric/protoutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func sendTransactionToPeer(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	userOrg string,
	channel string,
	ccName string,
	args ...string,
) (*gexec.Session, error) {
	return network.PeerUserSession(peer, userOrg, commands.ChaincodeInvoke{
		ChannelID: channel,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      ccName,
		Ctor:      cmn.CtorFromSlice(args),
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", peer.Name), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})

}

func invokeNBTx(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	userOrg string,
	channel string,
	ccName string,
	args ...string,
) *types.InvokeResult {
	result := &types.InvokeResult{}
	sess, err := sendTransactionToPeer(network, peer, orderer, userOrg, channel, ccName, args...)
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit())
	result.SetResponse(sess.Out.Contents())
	result.SetMessage(sess.Err.Contents())
	result.SetErrorCode(int32(sess.ExitCode()))

	if err == nil {
		Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
		Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))
	}

	return result
}

// Deprecated: need to remove after migrating to testsuite
func invokeNBTxWithCheckErr(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	userOrg string,
	checkErr CheckResultFunc,
	channel string,
	ccName string,
	args ...string,
) {
	sess, err := sendTransactionToPeer(network, peer, orderer, userOrg, channel, ccName, args...)
	if checkErr != nil {
		Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit())
		res := checkErr(err, sess.ExitCode(), sess.Err.Contents(), sess.Out.Contents())
		Expect(res).Should(BeEmpty())

		return
	}

	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))
}

// Deprecated: need to remove after migrating to testsuite
// NBTxInvoke func for invoke to foundation fabric
func NBTxInvoke(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	checkErr CheckResultFunc,
	channel string,
	ccName string,
	args ...string,
) {
	invokeNBTxWithCheckErr(network, peer, orderer, "User1", checkErr, channel, ccName, args...)
}

// Deprecated: need to remove after migrating to testsuite
// NBTxInvokeByRobot func for invoke to foundation fabric from robot
func NBTxInvokeByRobot(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	checkErr CheckResultFunc,
	channel string,
	ccName string,
	args ...string,
) {
	invokeNBTxWithCheckErr(network, peer, orderer, "User2", checkErr, channel, ccName, args...)
}

// Deprecated: need to remove after migrating to testsuite
// NBTxInvokeWithSign func for invoke with sign to foundation fabric
func NBTxInvokeWithSign(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	checkErr CheckResultFunc,
	channel string,
	ccName string,
	user *UserFoundation,
	fn string,
	requestID string,
	nonce string,
	args ...string,
) {
	ctorArgs := append(append([]string{fn, requestID, channel, ccName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	invokeNBTxWithCheckErr(network, peer, orderer, "User1", checkErr, channel, ccName, ctorArgs...)
}

func batchResponseProcess(response *pbfound.TxResponse, txID string, isValid bool, checkErr CheckResultFunc) bool {
	if hex.EncodeToString(response.GetId()) == txID {
		Expect(isValid).To(BeTrue())
		if checkErr != nil {
			Expect(response.GetError()).NotTo(BeNil())
			res := checkErr(nil, 1, []byte(response.GetError().GetError()), nil)
			Expect(res).Should(BeEmpty())
			return true
		}
		Expect(response.GetError()).To(BeNil())
		return true
	}

	return false
}

func deliverResponseProcess(resp *pb.DeliverResponse, txID string, checkErr CheckResultFunc) bool {
	b, ok := resp.GetType().(*pb.DeliverResponse_Block)
	Expect(ok).To(BeTrue())

	txFilter := b.Block.GetMetadata().GetMetadata()[cb.BlockMetadataIndex_TRANSACTIONS_FILTER]
	for txIndex, ebytes := range b.Block.GetData().GetData() {
		if ebytes == nil {
			continue
		}

		isValid := true
		if len(txFilter) != 0 &&
			pb.TxValidationCode(txFilter[txIndex]) != pb.TxValidationCode_VALID {
			isValid = false
		}

		env, err := protoutil.GetEnvelopeFromBlock(ebytes)
		if err != nil {
			continue
		}

		// get the payload from the envelope
		payload, err := protoutil.UnmarshalPayload(env.GetPayload())
		Expect(err).NotTo(HaveOccurred())

		if payload.GetHeader() == nil {
			continue
		}

		chdr, err := protoutil.UnmarshalChannelHeader(payload.GetHeader().GetChannelHeader())
		Expect(err).NotTo(HaveOccurred())

		if cb.HeaderType(chdr.GetType()) != cb.HeaderType_ENDORSER_TRANSACTION {
			continue
		}

		tx, err := protoutil.UnmarshalTransaction(payload.GetData())
		Expect(err).NotTo(HaveOccurred())

		for _, action := range tx.GetActions() {
			chaincodeActionPayload, err := protoutil.UnmarshalChaincodeActionPayload(action.GetPayload())
			Expect(err).NotTo(HaveOccurred())

			if chaincodeActionPayload.GetAction() == nil {
				continue
			}

			propRespPayload, err := protoutil.UnmarshalProposalResponsePayload(chaincodeActionPayload.GetAction().GetProposalResponsePayload())
			Expect(err).NotTo(HaveOccurred())

			caPayload, err := protoutil.UnmarshalChaincodeAction(propRespPayload.GetExtension())
			Expect(err).NotTo(HaveOccurred())

			ccEvent, err := protoutil.UnmarshalChaincodeEvents(caPayload.GetEvents())
			Expect(err).NotTo(HaveOccurred())

			if ccEvent.GetEventName() == "batchExecute" {
				batchResponse := &pbfound.BatchResponse{}
				err = proto.Unmarshal(caPayload.GetResponse().GetPayload(), batchResponse)
				Expect(err).NotTo(HaveOccurred())

				for _, r := range batchResponse.GetTxResponses() {
					if batchResponseProcess(r, txID, isValid, checkErr) {
						return true
					}
				}
			}
		}
	}

	return false
}

// Deprecated: need to remove after migrating to testsuite
func invokeTxWithCheckErr(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	userOrg string,
	channel string,
	ccName string,
	checkErr CheckResultFunc,
	args ...string,
) (txID string) {
	lh := nwo.GetLedgerHeight(network, peer, channel)

	By("send transaction to peer")
	sess, err := sendTransactionToPeer(network, peer, orderer, userOrg, channel, ccName, args...)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	l := sess.Err.Contents()
	txID = scanTxIDInLog(l)
	Expect(txID).NotTo(BeEmpty())

	By("getting the signer for user1 on peer " + peer.ID())
	signer := network.PeerUserSigner(peer, "User1")

	By("creating the deliver client to peer " + peer.ID())
	pcc := network.PeerClientConn(peer)
	defer func() {
		err := pcc.Close()
		Expect(err).NotTo(HaveOccurred())
	}()
	ctx, cancel := context.WithTimeout(context.Background(), network.EventuallyTimeout)
	defer cancel()
	dc, err := pb.NewDeliverClient(pcc).Deliver(ctx)
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := dc.CloseSend()
		Expect(err).NotTo(HaveOccurred())
	}()

	By("starting filtered delivery on peer " + peer.ID())
	deliverEnvelope, err := protoutil.CreateSignedEnvelope(
		cb.HeaderType_DELIVER_SEEK_INFO,
		channel,
		signer,
		&ab.SeekInfo{
			Behavior: ab.SeekInfo_BLOCK_UNTIL_READY,
			Start: &ab.SeekPosition{
				Type: &ab.SeekPosition_Specified{
					Specified: &ab.SeekSpecified{Number: uint64(lh)},
				},
			},
			Stop: &ab.SeekPosition{
				Type: &ab.SeekPosition_Specified{
					Specified: &ab.SeekSpecified{Number: math.MaxUint64},
				},
			},
		},
		0,
		0,
	)
	Expect(err).NotTo(HaveOccurred())
	err = dc.Send(deliverEnvelope)
	Expect(err).NotTo(HaveOccurred())

	By("waiting for deliver event on peer " + peer.ID())
	for {
		resp, err := dc.Recv()
		Expect(err).NotTo(HaveOccurred())

		if deliverResponseProcess(resp, txID, checkErr) {
			return txID
		}
	}
}

func invokeTx(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	userOrg string,
	channel string,
	ccName string,
	args ...string,
) *types.InvokeResult {
	result := &types.InvokeResult{}
	lh := nwo.GetLedgerHeight(network, peer, channel)

	sess, err := sendTransactionToPeer(network, peer, orderer, userOrg, channel, ccName, args...)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	l := sess.Err.Contents()
	txID := scanTxIDInLog(l)
	Expect(txID).NotTo(BeEmpty())
	result.SetTxID(txID)

	By("getting the signer for user1 on peer " + peer.ID())
	signer := network.PeerUserSigner(peer, "User1")

	By("creating the deliver client to peer " + peer.ID())
	pcc := network.PeerClientConn(peer)
	defer func() {
		err := pcc.Close()
		Expect(err).NotTo(HaveOccurred())
	}()
	ctx, cancel := context.WithTimeout(context.Background(), network.EventuallyTimeout)
	defer cancel()
	dc, err := pb.NewDeliverClient(pcc).Deliver(ctx)
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := dc.CloseSend()
		Expect(err).NotTo(HaveOccurred())
	}()

	By("starting filtered delivery on peer " + peer.ID())
	deliverEnvelope, err := protoutil.CreateSignedEnvelope(
		cb.HeaderType_DELIVER_SEEK_INFO,
		channel,
		signer,
		&ab.SeekInfo{
			Behavior: ab.SeekInfo_BLOCK_UNTIL_READY,
			Start: &ab.SeekPosition{
				Type: &ab.SeekPosition_Specified{
					Specified: &ab.SeekSpecified{Number: uint64(lh)},
				},
			},
			Stop: &ab.SeekPosition{
				Type: &ab.SeekPosition_Specified{
					Specified: &ab.SeekSpecified{Number: math.MaxUint64},
				},
			},
		},
		0,
		0,
	)
	Expect(err).NotTo(HaveOccurred())
	err = dc.Send(deliverEnvelope)
	Expect(err).NotTo(HaveOccurred())

	By("waiting for deliver event on peer " + peer.ID())
	for {
		resp, err := dc.Recv()
		Expect(err).NotTo(HaveOccurred())

		b, ok := resp.GetType().(*pb.DeliverResponse_Block)
		Expect(ok).To(BeTrue())

		txFilter := b.Block.GetMetadata().GetMetadata()[cb.BlockMetadataIndex_TRANSACTIONS_FILTER]
		for txIndex, ebytes := range b.Block.GetData().GetData() {
			var env *cb.Envelope

			if ebytes == nil {
				continue
			}

			isValid := true
			if len(txFilter) != 0 &&
				pb.TxValidationCode(txFilter[txIndex]) != pb.TxValidationCode_VALID {
				isValid = false
			}

			env, err = protoutil.GetEnvelopeFromBlock(ebytes)
			if err != nil {
				continue
			}

			// get the payload from the envelope
			payload, err := protoutil.UnmarshalPayload(env.GetPayload())
			Expect(err).NotTo(HaveOccurred())

			if payload.GetHeader() == nil {
				continue
			}

			chdr, err := protoutil.UnmarshalChannelHeader(payload.GetHeader().GetChannelHeader())
			Expect(err).NotTo(HaveOccurred())

			if cb.HeaderType(chdr.GetType()) != cb.HeaderType_ENDORSER_TRANSACTION {
				continue
			}

			tx, err := protoutil.UnmarshalTransaction(payload.GetData())
			Expect(err).NotTo(HaveOccurred())

			for _, action := range tx.GetActions() {
				chaincodeActionPayload, err := protoutil.UnmarshalChaincodeActionPayload(action.GetPayload())
				Expect(err).NotTo(HaveOccurred())

				if chaincodeActionPayload.GetAction() == nil {
					continue
				}

				propRespPayload, err := protoutil.UnmarshalProposalResponsePayload(chaincodeActionPayload.GetAction().GetProposalResponsePayload())
				Expect(err).NotTo(HaveOccurred())

				caPayload, err := protoutil.UnmarshalChaincodeAction(propRespPayload.GetExtension())
				Expect(err).NotTo(HaveOccurred())

				ccEvent, err := protoutil.UnmarshalChaincodeEvents(caPayload.GetEvents())
				Expect(err).NotTo(HaveOccurred())

				if ccEvent.GetEventName() == "batchExecute" {
					batchResponse := &pbfound.BatchResponse{}
					err = proto.Unmarshal(caPayload.GetResponse().GetPayload(), batchResponse)
					Expect(err).NotTo(HaveOccurred())

					for _, r := range batchResponse.GetTxResponses() {
						if hex.EncodeToString(r.GetId()) == txID {
							Expect(isValid).To(BeTrue())
							if r.GetError() != nil {
								result.SetMessage([]byte(r.GetError().GetError()))
								result.SetErrorCode(r.GetError().GetCode())
							}
							return result
						}
					}
				}
			}
		}
	}
}

// Deprecated: need to remove after migrating to testsuite
// TxInvoke func for invoke to foundation fabric
func TxInvoke(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	channel string,
	ccName string,
	checkErr CheckResultFunc,
	args ...string,
) (txID string) {
	return invokeTxWithCheckErr(network, peer, orderer, "User1", channel, ccName, checkErr, args...)
}

// Deprecated: need to remove after migrating to testsuite
// TxInvokeByRobot func for invoke to foundation fabric from robot
func TxInvokeByRobot(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	channel string,
	ccName string,
	checkErr CheckResultFunc,
	args ...string,
) (txID string) {
	return invokeTxWithCheckErr(network, peer, orderer, "User2", channel, ccName, checkErr, args...)
}

// Deprecated: need to remove after migrating to testsuite
// TxInvokeWithSign func for invoke with sign to foundation fabric
func TxInvokeWithSign(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	channel string,
	ccName string,
	user *UserFoundation,
	fn string,
	requestID string,
	nonce string,
	checkErr CheckResultFunc,
	args ...string,
) (txID string) {
	ctorArgs := append(append([]string{fn, requestID, channel, ccName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	return invokeTxWithCheckErr(network, peer, orderer, "User1", channel, ccName, checkErr, ctorArgs...)
}

// Deprecated: need to remove after migrating to testsuite
// TxInvokeWithMultisign invokes transaction to foundation fabric with multisigned user
func TxInvokeWithMultisign(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	channel string,
	ccName string,
	user *UserFoundationMultisigned,
	fn string,
	requestID string,
	nonce string,
	checkErr CheckResultFunc,
	args ...string,
) (txID string) {
	ctorArgs := append(append([]string{fn, requestID, channel, ccName}, args...), nonce)
	pubKey, sMsgsByte, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	var sMsgsStr []string
	for _, sMsgByte := range sMsgsByte {
		sMsgsStr = append(sMsgsStr, base58.Encode(sMsgByte))
	}

	ctorArgs = append(append(ctorArgs, pubKey...), sMsgsStr...)
	return invokeTxWithCheckErr(network, peer, orderer, "User1", channel, ccName, checkErr, ctorArgs...)
}

func scanTxIDInLog(data []byte) string {
	// find: txid [......] committed with status
	re := regexp.MustCompile(fmt.Sprintf("txid \\[.*\\] committed with status"))
	loc := re.FindIndex(data)
	Expect(len(loc)).To(BeNumerically(">", 0))

	start := loc[0]
	_, data, ok := bytes.Cut(data[start:], []byte("["))
	Expect(ok).To(BeTrue())

	data, _, ok = bytes.Cut(data, []byte("]"))
	Expect(ok).To(BeTrue())

	return string(data)
}

func (ts *testSuite) TxInvoke(channelName, chaincodeName string, args ...string) *types.InvokeResult {
	return invokeTx(ts.network, ts.peer, ts.orderer, ts.mainUserName, channelName, chaincodeName, args...)
}

func (ts *testSuite) TxInvokeByRobot(channelName, chaincodeName string, args ...string) *types.InvokeResult {
	return invokeTx(ts.network, ts.peer, ts.orderer, ts.robotUserName, channelName, chaincodeName, args...)
}

func (ts *testSuite) TxInvokeWithSign(
	channelName string,
	chaincodeName string,
	user *UserFoundation,
	fn string,
	requestID string,
	nonce string,
	args ...string,
) *types.InvokeResult {
	ctorArgs := append(append([]string{fn, requestID, channelName, chaincodeName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	return ts.TxInvoke(channelName, chaincodeName, ctorArgs...)
}

func (ts *testSuite) TxInvokeWithMultisign(
	channelName string,
	chaincodeName string,
	user *UserFoundationMultisigned,
	fn string,
	requestID string,
	nonce string,
	args ...string,
) *types.InvokeResult {
	ctorArgs := append(append([]string{fn, requestID, channelName, chaincodeName}, args...), nonce)
	pubKey, sMsgsByte, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	var sMsgsStr []string
	for _, sMsgByte := range sMsgsByte {
		sMsgsStr = append(sMsgsStr, base58.Encode(sMsgByte))
	}

	ctorArgs = append(append(ctorArgs, pubKey...), sMsgsStr...)
	return ts.TxInvoke(channelName, chaincodeName, ctorArgs...)
}

func (ts *testSuite) NBTxInvoke(channelName, chaincodeName string, args ...string) *types.InvokeResult {
	return invokeNBTx(ts.network, ts.peer, ts.orderer, ts.mainUserName, channelName, chaincodeName, args...)
}

func (ts *testSuite) NBTxInvokeByRobot(channelName, chaincodeName string, args ...string) *types.InvokeResult {
	return invokeNBTx(ts.network, ts.peer, ts.orderer, ts.robotUserName, channelName, chaincodeName, args...)
}

func (ts *testSuite) NBTxInvokeWithSign(channelName, chaincodeName string, user *UserFoundation, fn, requestID, nonce string, args ...string) *types.InvokeResult {
	ctorArgs := append(append([]string{fn, requestID, channelName, chaincodeName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	return ts.NBTxInvoke(channelName, chaincodeName, ctorArgs...)
}
