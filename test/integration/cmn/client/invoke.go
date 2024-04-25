package client

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
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
	// TODO make sure that the transaction is executed by the robot
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
