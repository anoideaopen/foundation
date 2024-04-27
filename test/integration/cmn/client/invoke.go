package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strings"

	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
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

// NBTxInvoke func for invoke to foundation fabric
func NBTxInvoke(network *nwo.Network, peer *nwo.Peer, orderer *nwo.Orderer,
	channel string, ccName string, args ...string) {

	ctor := "\"" + strings.Join(args, "\", \"") + "\""
	sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
		ChannelID: channel,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      ccName,
		Ctor:      fmt.Sprintf(`{"Args":[%s]}`, ctor),
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))
}

// NBTxInvokeWithSign func for invoke with sign to foundation fabric
func NBTxInvokeWithSign(network *nwo.Network, peer *nwo.Peer, orderer *nwo.Orderer,
	channel string, ccName string, user *UserFoundation,
	fn string, requestID string, nonce string, args ...string) {
	ctorArgs := append(append([]string{fn, requestID, channel, ccName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	NBTxInvoke(network, peer, orderer, channel, ccName, ctorArgs...)
}

// TxInvoke func for invoke to foundation fabric
func TxInvoke(network *nwo.Network, peer *nwo.Peer, orderer *nwo.Orderer,
	channel string, ccName string, args ...string) {
	lh := nwo.GetLedgerHeight(network, peer, channel)

	ctor := "\"" + strings.Join(args, "\", \"") + "\""
	sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
		ChannelID: channel,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      ccName,
		Ctor:      fmt.Sprintf(`{"Args":[%s]}`, ctor),
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	l := sess.Err.Contents()
	txId := scanTxIDInLog(l)
	Expect(txId).NotTo(BeEmpty())

	By("getting the signer for user1 on peer " + peer.ID())
	signer := network.PeerUserSigner(peer, "User1")

	By("creating the deliver client to peer " + peer.ID())
	pcc := network.PeerClientConn(peer)
	defer pcc.Close()
	ctx, cancel := context.WithTimeout(context.Background(), network.EventuallyTimeout)
	defer cancel()
	dc, err := pb.NewDeliverClient(pcc).Deliver(ctx)
	Expect(err).NotTo(HaveOccurred())
	defer dc.CloseSend()

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

		b, ok := resp.Type.(*pb.DeliverResponse_Block)
		Expect(ok).To(BeTrue())

		txFilter := b.Block.GetMetadata().GetMetadata()[common.BlockMetadataIndex_TRANSACTIONS_FILTER]
		for txIndex, ebytes := range b.Block.GetData().GetData() {
			var env *common.Envelope

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
			payload, err := protoutil.UnmarshalPayload(env.Payload)
			Expect(err).NotTo(HaveOccurred())

			if payload.Header == nil {
				continue
			}

			chdr, err := protoutil.UnmarshalChannelHeader(payload.Header.ChannelHeader)
			Expect(err).NotTo(HaveOccurred())

			if common.HeaderType(chdr.GetType()) != common.HeaderType_ENDORSER_TRANSACTION {
				continue
			}

			tx, err := protoutil.UnmarshalTransaction(payload.GetData())
			Expect(err).NotTo(HaveOccurred())

			for _, action := range tx.GetActions() {
				chaincodeActionPayload, err := protoutil.UnmarshalChaincodeActionPayload(action.Payload)
				Expect(err).NotTo(HaveOccurred())

				if chaincodeActionPayload.Action == nil {
					continue
				}

				propRespPayload, err := protoutil.UnmarshalProposalResponsePayload(chaincodeActionPayload.Action.ProposalResponsePayload)
				Expect(err).NotTo(HaveOccurred())

				caPayload, err := protoutil.UnmarshalChaincodeAction(propRespPayload.Extension)
				Expect(err).NotTo(HaveOccurred())

				ccEvent, err := protoutil.UnmarshalChaincodeEvents(caPayload.Events)
				Expect(err).NotTo(HaveOccurred())

				if ccEvent.GetEventName() == "batchExecute" && isValid {
					batchResponse := &pbfound.BatchResponse{}
					err = proto.Unmarshal(caPayload.Response.Payload, batchResponse)
					Expect(err).NotTo(HaveOccurred())

					for _, r := range batchResponse.TxResponses {
						if hex.EncodeToString(r.GetId()) == txId {
							Expect(isValid).To(BeTrue())
							return
						}
					}
				}
			}
		}
	}
}

// TxInvokeWithSign func for invoke with sign to foundation fabric
func TxInvokeWithSign(network *nwo.Network, peer *nwo.Peer, orderer *nwo.Orderer,
	channel string, ccName string, user *UserFoundation,
	fn string, requestID string, nonce string, args ...string) {
	ctorArgs := append(append([]string{fn, requestID, channel, ccName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	TxInvoke(network, peer, orderer, channel, ccName, ctorArgs...)
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
