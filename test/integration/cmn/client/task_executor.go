package client

import (
	"encoding/json"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func ExecuteTask(user *UserFoundation, network *nwo.Network, channel string, chaincode string, method string, args []string) {
	requestID := time.Now().UTC().Format(time.RFC3339Nano)
	ctorArgs := append(append([]string{method, requestID, channel, chaincode}, args...), NewNonceByTime().Get())
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())
	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))

	taskID := time.Now().UTC().Format(time.RFC3339Nano)
	task := core.Task{
		ID:     taskID,
		Method: method,
		Args:   args,
	}
	tasks := []core.Task{task}
	ExecuteTasks(network, network.Peers[0], network.Orderers[0], user.UserID, nil, channel, chaincode, tasks...)
}

func ExecuteTasks(network *nwo.Network, peer *nwo.Peer, orderer *nwo.Orderer, userOrg string,
	checkErr CheckResultFunc, channel string, ccName string, tasks ...core.Task) {
	bytes, err := json.Marshal(core.ExecuteTasksRequest{Tasks: tasks})
	Expect(err).NotTo(HaveOccurred())

	sess, err := network.PeerUserSession(peer, userOrg, commands.ChaincodeInvoke{
		ChannelID: channel,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      ccName,
		Ctor:      cmn.CtorFromSlice([]string{core.ExecuteTasks, string(bytes)}),
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
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
