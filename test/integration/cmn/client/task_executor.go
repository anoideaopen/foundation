package client

import (
	"fmt"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"google.golang.org/protobuf/encoding/protojson"
)

func CreateTaskWithSignArgs(method string, channel string, chaincode string, user *UserFoundation, args ...string) (*proto.Task, error) {
	requestID := time.Now().UTC().Format(time.RFC3339Nano)

	args = append(append([]string{method, requestID, channel, chaincode}, args...), NewNonceByTime().Get())

	pubKey, sMsg, err := user.Sign(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to sign args: %w", err)
	}

	args = append(args, pubKey, base58.Encode(sMsg))

	taskID := time.Now().UTC().Format(time.RFC3339Nano)
	task := &proto.Task{
		Id:     taskID,
		Method: method,
		Args:   args[1:], // Exclude the method name from the args
	}

	return task, nil
}

func ExecuteTaskWithSign(
	network *nwo.Network,
	channel string,
	chaincode string,
	user *UserFoundation,
	checkErr CheckResultFunc,
	method string,
	args ...string,
) string {
	task, err := CreateTaskWithSignArgs(method, channel, chaincode, user, args...)
	if err != nil {
		panic(err)
	}
	txID := ExecuteTasks(network, network.Peers[0], network.Orderers[0], "User2", checkErr, channel, chaincode, task)
	return txID
}

func ExecuteTasks(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	userOrg string,
	checkErr CheckResultFunc,
	channel string,
	ccName string,
	tasks ...*proto.Task,
) string {
	bytes, err := protojson.Marshal(&proto.ExecuteTasksRequest{Tasks: tasks})
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

		return ""
	}

	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	l := sess.Err.Contents()
	txId := scanTxIDInLog(l)
	Expect(txId).NotTo(BeEmpty())
	return txId
}
