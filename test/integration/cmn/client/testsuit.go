package client

import (
	"encoding/hex"
	"fmt"
	"github.com/anoideaopen/acl/tests/common"
	pb "github.com/anoideaopen/foundation/proto"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
	"github.com/btcsuite/btcutil/base58"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	defaultOrg1Name      = "Org1"
	defaultOrg2Name      = "Org2"
	defaultMainUserName  = "User1"
	defaultRobotUserName = "User2"
	defaultPeerName      = "peer0"
)

type TestSuite interface {
	// Admin returns testsuite admin
	Admin() *UserFoundation
	// FeeSetter returns testsuite fee setter user
	FeeSetter() *UserFoundation
	// FeeAddressSetter returns testsuite fee address setter
	FeeAddressSetter() *UserFoundation
	// Network returns testsuite network
	Network() *nwo.Network
	// NetworkFound returns testsuite network foundation
	NetworkFound() *cmn.NetworkFoundation
	// Peer returns testsuite peer
	Peer() *nwo.Peer
	// DeployChannels deploys channels to testsuite network
	DeployChannels()
	// TxInvoke func for invoke to foundation fabric
	TxInvoke(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string) string
	// TxInvokeByRobot func for invoke to foundation fabric from robot
	TxInvokeByRobot(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string) string
	// TxInvokeWithSign func for invoke with sign to foundation fabric
	TxInvokeWithSign(channelName, chaincodeName string, user *UserFoundation, fn, requestID, nonce string, checkErr CheckResultFunc, args ...string) (txId string)
	// TxInvokeWithMultisign invokes transaction to foundation fabric with multisigned user
	TxInvokeWithMultisign(channelName, chaincodeName string, user *UserFoundationMultisigned, fn, requestID, nonce string, checkErr CheckResultFunc, args ...string) (txId string)
	// NBTxInvoke func for invoke to foundation fabric
	NBTxInvoke(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string)
	// NBTxInvokeByRobot func for invoke to foundation fabric from robot
	NBTxInvokeByRobot(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string)
	// NBTxInvokeWithSign func for invoke with sign to foundation fabric
	NBTxInvokeWithSign(channelName, chaincodeName string, checkErr CheckResultFunc, user *UserFoundation, fn, requestID, nonce string, args ...string)
	// Query func for query from foundation fabric
	Query(channelName, chaincodeName string, checkResultFunc CheckResultFunc, args ...string)
	// QueryWithSign func for query with sign from foundation fabric
	QueryWithSign(channelName, chaincodeName string, checkResultFunc CheckResultFunc, user *UserFoundation, fn, requestID, nonce string, args ...string)
	// AddUser adds new user to ACL channel
	AddUser(user *UserFoundation)
	// AddUserMultisigned adds multisigned user to ACL channel
	AddUserMultisigned(user *UserFoundationMultisigned)
	// AddRights adds right for defined user with specified role and operation to ACL channel
	AddRights(channelName, chaincodeName, role, operation string, user *UserFoundation)
	// RemoveRights removes right for defined user with specified role and operation to ACL channel
	RemoveRights(channelName, chaincodeName, role, operation string, user *UserFoundation)
}

type testSuite struct {
	network          *nwo.Network
	networkFound     *cmn.NetworkFoundation
	peer             *nwo.Peer
	orderer          *nwo.Orderer
	components       *nwo.Components
	testDir          string
	org1Name         string
	org2Name         string
	mainUserName     string
	robotUserName    string
	channels         []string
	admin            *UserFoundation
	feeSetter        *UserFoundation
	feeAddressSetter *UserFoundation
	skiBackend       string
	skiRobot         string
}

func initPeer(network *nwo.Network, orgName string) *nwo.Peer {
	return network.Peer(orgName, defaultPeerName)
}

func startPort(portRange integration.TestPortRange) int {
	return portRange.StartPortForNode()
}

func (ts *testSuite) initNetwork(
	redisDBAddress string,
	testDir string,
	dockerClient *docker.Client,
	testPort integration.TestPortRange,
	ordererProcesses []ifrit.Process,
	peerProcess ifrit.Process,
) {
	var ordererRunners []*ginkgomon.Runner

	networkConfig := nwo.MultiNodeSmartBFT()
	networkConfig.Channels = nil

	peerChannels := make([]*nwo.PeerChannel, 0, cap(ts.channels))
	for _, ch := range ts.channels {
		peerChannels = append(peerChannels, &nwo.PeerChannel{
			Name:   ch,
			Anchor: true,
		})
	}
	for _, peer := range networkConfig.Peers {
		peer.Channels = peerChannels
	}

	ts.network = nwo.New(networkConfig, testDir, dockerClient, startPort(testPort), ts.components)

	cwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	ts.network.ExternalBuilders = append(ts.network.ExternalBuilders,
		fabricconfig.ExternalBuilder{
			Path:                 filepath.Join(cwd, ".", "externalbuilders", "binary"),
			Name:                 "binary",
			PropagateEnvironment: []string{"GOPROXY"},
		},
	)

	ts.networkFound = cmn.New(ts.network, ts.channels)
	ts.networkFound.Robot.RedisAddresses = []string{redisDBAddress}

	ts.networkFound.GenerateConfigTree()
	ts.networkFound.Bootstrap()

	for _, orderer := range ts.network.Orderers {
		runner := ts.network.OrdererRunner(orderer)
		runner.Command.Env = append(runner.Command.Env, "FABRIC_LOGGING_SPEC=orderer.consensus.smartbft=debug:grpc=debug")
		ordererRunners = append(ordererRunners, runner)
		proc := ifrit.Invoke(runner)
		ordererProcesses = append(ordererProcesses, proc)
		Eventually(proc.Ready(), ts.network.EventuallyTimeout).Should(BeClosed())
	}

	peerGroupRunner, _ := fabricnetwork.PeerGroupRunners(ts.network)
	peerProcess = ifrit.Invoke(peerGroupRunner)
	Eventually(peerProcess.Ready(), ts.network.EventuallyTimeout).Should(BeClosed())

	ts.peer = initPeer(ts.network, ts.org1Name)
	ts.orderer = ts.network.Orderers[0]

	By("Joining orderers to channels")
	for _, channel := range ts.channels {
		fabricnetwork.JoinChannel(ts.network, channel)
	}

	By("Waiting for followers to see the leader")
	Eventually(ordererRunners[1].Err(), ts.network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
	Eventually(ordererRunners[2].Err(), ts.network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
	Eventually(ordererRunners[3].Err(), ts.network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))

	By("Joining peers to channels")
	for _, channel := range ts.channels {
		ts.network.JoinChannel(channel, ts.orderer, ts.network.PeersWithChannel(channel)...)
	}

	pathToPrivateKeyBackend := ts.network.PeerUserKey(ts.peer, ts.mainUserName)
	skiBackend, err := cmn.ReadSKI(pathToPrivateKeyBackend)
	Expect(err).NotTo(HaveOccurred())

	pathToPrivateKeyRobot := ts.network.PeerUserKey(ts.peer, ts.robotUserName)
	skiRobot, err := cmn.ReadSKI(pathToPrivateKeyRobot)
	Expect(err).NotTo(HaveOccurred())

	ts.skiBackend = skiBackend
	ts.skiRobot = skiRobot

	admin, err := NewUserFoundation(pbfound.KeyType_ed25519)
	Expect(err).NotTo(HaveOccurred())
	Expect(admin.PrivateKeyBytes).NotTo(Equal(nil))

	feeSetter, err := NewUserFoundation(pbfound.KeyType_ed25519)
	Expect(err).NotTo(HaveOccurred())
	Expect(feeSetter.PrivateKeyBytes).NotTo(Equal(nil))

	feeAddressSetter, err := NewUserFoundation(pbfound.KeyType_ed25519)
	Expect(err).NotTo(HaveOccurred())
	Expect(feeAddressSetter.PrivateKeyBytes).NotTo(Equal(nil))
}

func NewTestSuite(
	org1Name string,
	org2Name string,
	mainUserName string,
	robotUserName string,
	redisDBAddress string,
	channels []string,
	testDir string,
	dockerClient *docker.Client,
	testPort integration.TestPortRange,
	components *nwo.Components,
	ordererProcesses []ifrit.Process,
	peerProcesses ifrit.Process,
) TestSuite {
	if org1Name == "" {
		org1Name = defaultOrg1Name
	}

	if org2Name == "" {
		org2Name = defaultOrg2Name
	}

	if mainUserName == "" {
		mainUserName = defaultMainUserName
	}

	if robotUserName == "" {
		robotUserName = defaultRobotUserName
	}

	ts := &testSuite{
		org1Name:      org1Name,
		org2Name:      org2Name,
		mainUserName:  mainUserName,
		robotUserName: robotUserName,
		channels:      channels,
		components:    components,
	}

	ts.initNetwork(redisDBAddress, testDir, dockerClient, testPort, ordererProcesses, peerProcesses)

	return ts
}

func (ts *testSuite) Admin() *UserFoundation {
	return ts.admin
}

func (ts *testSuite) FeeSetter() *UserFoundation {
	return ts.feeSetter
}

func (ts *testSuite) FeeAddressSetter() *UserFoundation {
	return ts.feeAddressSetter
}

func (ts *testSuite) Network() *nwo.Network {
	return ts.network
}

func (ts *testSuite) NetworkFound() *cmn.NetworkFoundation {
	return ts.networkFound
}

func (ts *testSuite) Peer() *nwo.Peer {
	return ts.peer
}

func (ts *testSuite) DeployChannels() {
	for _, channel := range ts.channels {
		switch channel {
		case cmn.ChannelAcl:
			cmn.DeployACL(ts.network, ts.components, ts.peer, ts.testDir, ts.skiBackend, ts.admin.PublicKeyBase58, ts.admin.KeyType)
		case cmn.ChannelFiat:
			cmn.DeployFiat(ts.network, ts.components, ts.peer, ts.testDir, ts.skiRobot, ts.admin.AddressBase58Check, ts.feeSetter.AddressBase58Check, ts.feeAddressSetter.AddressBase58Check)
		case cmn.ChannelCC:
			cmn.DeployCC(ts.network, ts.components, ts.peer, ts.testDir, ts.skiRobot, ts.admin.AddressBase58Check)
		case cmn.ChannelIndustrial:
			cmn.DeployIndustrial(ts.network, ts.components, ts.peer, ts.testDir, ts.skiRobot, ts.admin.AddressBase58Check, ts.feeSetter.AddressBase58Check, ts.feeAddressSetter.AddressBase58Check)
		default:
			continue
		}
	}
}

func (ts *testSuite) TxInvoke(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string) string {
	return invokeTx(ts.network, ts.peer, ts.orderer, ts.mainUserName, channelName, chaincodeName, checkErr, args...)
}

func (ts *testSuite) TxInvokeByRobot(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string) string {
	return invokeTx(ts.network, ts.peer, ts.orderer, ts.robotUserName, channelName, chaincodeName, checkErr, args...)
}

func (ts *testSuite) TxInvokeWithSign(
	channelName string,
	chaincodeName string,
	user *UserFoundation,
	fn string,
	requestID string,
	nonce string,
	checkErr CheckResultFunc,
	args ...string,
) (txId string) {
	ctorArgs := append(append([]string{fn, requestID, channelName, chaincodeName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	return ts.TxInvoke(channelName, chaincodeName, checkErr, ctorArgs...)
}

func (ts *testSuite) TxInvokeWithMultisign(
	channelName string,
	chaincodeName string,
	user *UserFoundationMultisigned,
	fn string,
	requestID string,
	nonce string,
	checkErr CheckResultFunc,
	args ...string,
) (txId string) {
	ctorArgs := append(append([]string{fn, requestID, channelName, chaincodeName}, args...), nonce)
	pubKey, sMsgsByte, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	var sMsgsStr []string
	for _, sMsgByte := range sMsgsByte {
		sMsgsStr = append(sMsgsStr, base58.Encode(sMsgByte))
	}

	ctorArgs = append(append(ctorArgs, pubKey...), sMsgsStr...)
	return ts.TxInvoke(channelName, chaincodeName, checkErr, ctorArgs...)
}

func (ts *testSuite) NBTxInvoke(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string) {
	invokeNBTx(ts.network, ts.peer, ts.orderer, ts.mainUserName, checkErr, channelName, chaincodeName, args...)
}

func (ts *testSuite) NBTxInvokeByRobot(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string) {
	invokeNBTx(ts.network, ts.peer, ts.orderer, ts.robotUserName, checkErr, channelName, chaincodeName, args...)
}

func (ts *testSuite) NBTxInvokeWithSign(channelName, chaincodeName string, checkErr CheckResultFunc, user *UserFoundation, fn, requestID, nonce string, args ...string) {
	ctorArgs := append(append([]string{fn, requestID, channelName, chaincodeName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	ts.NBTxInvoke(channelName, chaincodeName, checkErr, ctorArgs...)
}

// Query func for query from foundation fabric
func (ts *testSuite) Query(channelName, chaincodeName string, checkResultFunc CheckResultFunc, args ...string) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: channelName,
			Name:      chaincodeName,
			Ctor:      cmn.CtorFromSlice(args),
		})
		Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit())

		return checkResultFunc(err, sess.ExitCode(), sess.Err.Contents(), sess.Out.Contents())
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

// QueryWithSign func for query with sign from foundation fabric
func (ts *testSuite) QueryWithSign(
	channelName string,
	chaincodeName string,
	checkResultFunc CheckResultFunc,
	user *UserFoundation,
	fn string,
	requestID string,
	nonce string,
	args ...string,
) {
	ctorArgs := append(append([]string{fn, requestID, channelName, chaincodeName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	ts.Query(channelName, chaincodeName, checkResultFunc, ctorArgs...)
}

// AddUser adds new user to ACL channel
func (ts *testSuite) AddUser(user *UserFoundation) {
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor: cmn.CtorFromSlice(
			[]string{
				"addUserWithPublicKeyType",
				user.PublicKeyBase58,
				"test",
				user.UserID,
				"true",
				user.KeyType.String(),
			},
		),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckUser(user)
}

// AddUserMultisigned adds multisigned user
func (ts *testSuite) AddUserMultisigned(user *UserFoundationMultisigned) {
	ctorArgs := []string{common.FnAddMultisig, strconv.Itoa(len(user.Users)), NewNonceByTime().Get()}
	publicKeys, sMsgsByte, err := user.Sign(ctorArgs...)
	var sMsgsStr []string
	for _, sMsgByte := range sMsgsByte {
		sMsgsStr = append(sMsgsStr, hex.EncodeToString(sMsgByte))
	}
	ctorArgs = append(append(ctorArgs, publicKeys...), sMsgsStr...)
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice(ctorArgs),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckUserMultisigned(user)
}

func (ts *testSuite) CheckUser(user *UserFoundation) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{"checkKeys", user.PublicKeyBase58}),
		})
		Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit())
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		resp := &pb.AclResponse{}
		err = proto.Unmarshal(out, resp)
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		addr := base58.CheckEncode(resp.GetAddress().GetAddress().GetAddress()[1:], resp.GetAddress().GetAddress().GetAddress()[0])
		if addr != user.AddressBase58Check {
			return fmt.Sprintf("Error: expected %s, received %s", user.AddressBase58Check, addr)
		}

		return ""
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

func (ts *testSuite) CheckUserMultisigned(user *UserFoundationMultisigned) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{common.FnCheckKeys, user.PublicKey()}),
		})
		Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit())
		Expect(sess.ExitCode()).To(Equal(0))
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		resp := &pb.AclResponse{}
		err = proto.Unmarshal(out, resp)
		Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		addressBytes := resp.GetAddress().GetAddress().GetAddress()
		addr := base58.CheckEncode(addressBytes[1:], addressBytes[0])
		if addr != user.AddressBase58Check {
			return fmt.Sprintf("Error: expected %s, received %s", user.AddressBase58Check, addr)
		}

		return ""
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

func (ts *testSuite) AddRights(channelName, chaincodeName, role, operation string, user *UserFoundation) {
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice([]string{"addRights", channelName, chaincodeName, role, operation, user.AddressBase58Check}),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckRights(channelName, chaincodeName, role, operation, user, true)
}

func (ts *testSuite) RemoveRights(channelName, chaincodeName, role, operation string, user *UserFoundation) {
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice([]string{"removeRights", channelName, chaincodeName, role, operation, user.AddressBase58Check}),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckRights(channelName, chaincodeName, role, operation, user, false)
}

func (ts *testSuite) CheckRights(channelName, chaincodeName, role, operation string, user *UserFoundation, result bool) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{"getAccountOperationRightJSON", channelName, chaincodeName, role, operation, user.AddressBase58Check}),
		})
		Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit())
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		haveRight := &pb.HaveRight{}
		err = protojson.Unmarshal(out, haveRight)
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		if haveRight.HaveRight != result {
			return fmt.Sprintf("Error: expected %t, received %t", result, haveRight.HaveRight)
		}

		return ""
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}
