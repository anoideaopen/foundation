package client

import (
	"fmt"
	"github.com/anoideaopen/foundation/mocks"
	"time"

	"github.com/anoideaopen/foundation/core"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client/types"
	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"google.golang.org/protobuf/encoding/protojson"
)

func newTaskID() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func createTask(method string, args ...string) *pbfound.Task {
	return &pbfound.Task{
		Id:     newTaskID(),
		Method: method,
		Args:   args,
	}
}

func CreateTaskWithSignArgs(method string, channel string, chaincode string, user *mocks.UserFoundation, args ...string) (*pbfound.Task, error) {
	requestID := time.Now().UTC().Format(time.RFC3339Nano)

	args = append(append([]string{method, requestID, channel, chaincode}, args...), NewNonceByTime().Get())

	pubKey, sMsg, err := user.Sign(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to sign args: %w", err)
	}

	args = append(args, pubKey, base58.Encode(sMsg))

	task := createTask(method, args[1:]...) // Exclude the method name from the args

	return task, nil
}

func (ts *FoundationTestSuite) executeTasks(
	channel string,
	ccName string,
	tasks ...*pbfound.Task,
) *types.InvokeResult {
	result := &types.InvokeResult{}
	bytes, err := protojson.Marshal(&pbfound.ExecuteTasksRequest{Tasks: tasks})
	Expect(err).NotTo(HaveOccurred())

	sess, err := ts.Network.PeerUserSession(ts.Peer, ts.RobotUserName, commands.ChaincodeInvoke{
		ChannelID: channel,
		Orderer:   ts.Network.OrdererAddress(ts.Orderer, nwo.ListenPort),
		Name:      ccName,
		Ctor:      cmn.CtorFromSlice([]string{core.ExecuteTasks, string(bytes)}),
		PeerAddresses: []string{
			ts.Network.PeerAddress(ts.Network.Peer(ts.Org1Name, ts.Peer.Name), nwo.ListenPort),
			ts.Network.PeerAddress(ts.Network.Peer(ts.Org2Name, ts.Peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})

	Eventually(sess, ts.Network.EventuallyTimeout).Should(gexec.Exit())
	Expect(sess).NotTo(BeNil())
	result.SetResponse(sess.Out.Contents())
	result.SetMessage(sess.Err.Contents())
	result.SetErrorCode(int32(sess.ExitCode()))

	if err == nil {
		Eventually(sess, ts.Network.EventuallyTimeout).Should(gexec.Exit(0))
		Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))
	}

	l := sess.Err.Contents()
	txID := scanTxIDInLog(l)
	Expect(txID).NotTo(BeEmpty())

	return result
}

func (ts *FoundationTestSuite) ExecuteTask(channel string, chaincode string, method string, args ...string) string {
	task := createTask(method, args...)
	return ts.ExecuteTasks(channel, chaincode, task)
}

func (ts *FoundationTestSuite) ExecuteTasks(channel string, chaincode string, tasks ...*pbfound.Task) string {
	return ts.executeTasks(
		channel,
		chaincode,
		tasks...,
	).TxID()
}

func (ts *FoundationTestSuite) ExecuteTaskWithSign(channel string, chaincode string, user *mocks.UserFoundation, method string, args ...string) string {
	task, err := CreateTaskWithSignArgs(method, channel, chaincode, user, args...)
	if err != nil {
		panic(err)
	}

	return ts.ExecuteTasks(channel, chaincode, task)
}
