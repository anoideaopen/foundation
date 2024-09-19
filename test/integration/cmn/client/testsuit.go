package client

import (
	"os"
	"path/filepath"
	"syscall"
	"time"

	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	"github.com/anoideaopen/robot/helpers/ntesting"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
)

const (
	defaultOrg1Name      = "Org1"
	defaultOrg2Name      = "Org2"
	defaultMainUserName  = "User1"
	defaultRobotUserName = "User2"
	defaultPeerName      = "peer0"
)

type testSuite struct {
	components          *nwo.Components
	options             *networkOptions
	testDir             string
	dockerClient        *docker.Client
	network             *nwo.Network
	networkFound        *cmn.NetworkFoundation
	peer                *nwo.Peer
	orderer             *nwo.Orderer
	redisDB             *runner.RedisDB
	redisProcess        ifrit.Process
	robotProc           ifrit.Process
	ordererProcesses    []ifrit.Process
	peerProcess         ifrit.Process
	channelTransferProc ifrit.Process
	peerRunner          ifrit.Runner
	ordererRunners      []*ginkgomon.Runner
	org1Name            string
	org2Name            string
	mainUserName        string
	robotUserName       string
	admin               *UserFoundation
	feeSetter           *UserFoundation
	feeAddressSetter    *UserFoundation
	skiBackend          string
	skiRobot            string

	isInit bool
}

func NewTestSuite(components *nwo.Components) TestSuite {
	testDir, err := os.MkdirTemp("", "foundation")
	Expect(err).NotTo(HaveOccurred())

	dockerClient, err := docker.NewClientFromEnv()
	Expect(err).NotTo(HaveOccurred())

	ts := &testSuite{
		org1Name:      defaultOrg1Name,
		org2Name:      defaultOrg2Name,
		mainUserName:  defaultMainUserName,
		robotUserName: defaultRobotUserName,
		components:    components,
		testDir:       testDir,
		dockerClient:  dockerClient,
		// networkProcess:   nil,
		ordererProcesses: nil,
		peerProcess:      nil,
		options: &networkOptions{
			RobotCfg:           cmn.RobotCfgDefault,
			ChannelTransferCfg: cmn.ChannelTransferCfgDefault,
			Templates: &cmn.TemplatesFound{
				Robot:           "",
				ChannelTransfer: "",
			},
		},
		isInit: false,
	}

	return ts
}

func (ts *testSuite) InitNetwork(channels []string, testPort integration.TestPortRange, opts ...NetworkOption) {
	ts.options.Channels = channels
	ts.options.TestPort = testPort

	for _, opt := range opts {
		err := opt(ts.options)
		Expect(err).NotTo(HaveOccurred())
	}
	Expect(ts.options.Channels).NotTo(BeEmpty())
	Expect(ts.options.TestPort).NotTo(BeNil())

	networkConfig := nwo.MultiNodeSmartBFT()
	networkConfig.Channels = nil

	peerChannels := make([]*nwo.PeerChannel, 0, cap(ts.options.Channels))
	for _, ch := range ts.options.Channels {
		peerChannels = append(peerChannels, &nwo.PeerChannel{
			Name:   ch,
			Anchor: true,
		})
	}
	for _, peer := range networkConfig.Peers {
		peer.Channels = peerChannels
	}

	ts.network = nwo.New(networkConfig, ts.testDir, ts.dockerClient, ts.options.TestPort.StartPortForNode(), ts.components)

	cwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	ts.network.ExternalBuilders = append(ts.network.ExternalBuilders,
		fabricconfig.ExternalBuilder{
			Path:                 filepath.Join(cwd, ".", "externalbuilders", "binary"),
			Name:                 "binary",
			PropagateEnvironment: []string{"GOPROXY"},
		},
	)

	ts.networkFound = cmn.New(
		ts.network,
		ts.options.Channels,
		cmn.WithRobotCfg(ts.options.RobotCfg),
		cmn.WithChannelTransferCfg(ts.options.ChannelTransferCfg),
		cmn.WithRobotTemplate(ts.options.Templates.Robot),
		cmn.WithChannelTransferTemplate(ts.options.Templates.ChannelTransfer),
	)

	if ts.redisDB != nil {
		ts.networkFound.Robot.RedisAddresses = []string{ts.redisDB.Address()}
		ts.networkFound.ChannelTransfer.RedisAddresses = []string{ts.redisDB.Address()}
	}

	ts.networkFound.GenerateConfigTree()
	ts.networkFound.Bootstrap()

	for _, orderer := range ts.network.Orderers {
		ordererRunner := ts.network.OrdererRunner(orderer)
		ordererRunner.Command.Env = append(ordererRunner.Command.Env, "FABRIC_LOGGING_SPEC=orderer.consensus.smartbft=debug:grpc=debug")
		ts.ordererRunners = append(ts.ordererRunners, ordererRunner)
		proc := ifrit.Invoke(ordererRunner)
		ts.ordererProcesses = append(ts.ordererProcesses, proc)
		Eventually(proc.Ready(), ts.network.EventuallyTimeout).Should(BeClosed())
	}

	peerGroupRunner, _ := fabricnetwork.PeerGroupRunners(ts.network)
	ts.peerProcess = ifrit.Invoke(peerGroupRunner)
	Eventually(ts.peerProcess.Ready(), ts.network.EventuallyTimeout).Should(BeClosed())

	ts.peer = ts.network.Peer(ts.org1Name, defaultPeerName)
	ts.orderer = ts.network.Orderers[0]

	By("Joining orderers to channels")
	for _, channel := range ts.options.Channels {
		fabricnetwork.JoinChannel(ts.network, channel)
	}

	By("Waiting for followers to see the leader")
	Eventually(ts.ordererRunners[1].Err(), ts.network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
	Eventually(ts.ordererRunners[2].Err(), ts.network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
	Eventually(ts.ordererRunners[3].Err(), ts.network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))

	By("Joining peers to channels")
	for _, channel := range ts.options.Channels {
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

	ts.admin, err = NewUserFoundation(pbfound.KeyType_ed25519)
	Expect(err).NotTo(HaveOccurred())
	Expect(ts.admin.PrivateKeyBytes).NotTo(Equal(nil))

	ts.feeSetter, err = NewUserFoundation(pbfound.KeyType_ed25519)
	Expect(err).NotTo(HaveOccurred())
	Expect(ts.feeSetter.PrivateKeyBytes).NotTo(Equal(nil))

	ts.feeAddressSetter, err = NewUserFoundation(pbfound.KeyType_ed25519)
	Expect(err).NotTo(HaveOccurred())
	Expect(ts.feeAddressSetter.PrivateKeyBytes).NotTo(Equal(nil))

	ts.isInit = true
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

func (ts *testSuite) TestDir() string {
	return ts.testDir
}

func (ts *testSuite) DockerClient() *docker.Client {
	return ts.dockerClient
}

func (ts *testSuite) CiData(opts ...CiDataOption) ntesting.CiTestData {
	Expect(ts.isInit).To(BeTrue())
	Expect(ts.redisDB).NotTo(BeNil())

	// setting default values
	ciData := ntesting.CiTestData{
		RedisAddr:             ts.redisDB.Address(),
		RedisPass:             "",
		HlfProfilePath:        ts.networkFound.ConnectionPath(ts.mainUserName),
		HlfFiatChannel:        cmn.ChannelFiat,
		HlfCcChannel:          cmn.ChannelCC,
		HlfIndustrialChannel:  cmn.ChannelIndustrial,
		HlfNoCcChannel:        "",
		HlfUserName:           "backend",
		HlfCert:               ts.network.PeerUserKey(ts.peer, ts.mainUserName),
		HlfFiatOwnerKey:       ts.admin.PublicKeyBase58,
		HlfCcOwnerKey:         ts.admin.PublicKeyBase58,
		HlfIndustrialOwnerKey: ts.admin.PublicKeyBase58,
		HlfIndustrialGroup1:   "",
		HlfIndustrialGroup2:   "",
		HlfSk:                 ts.network.PeerUserKey(ts.peer, ts.mainUserName),
		HlfDoSwapTests:        false,
		HlfDoMultiSwapTests:   false,
	}

	// setting options
	for _, opt := range opts {
		err := opt(ciData)
		Expect(err).NotTo(HaveOccurred())
	}

	return ciData
}

func (ts *testSuite) StartRedis() {
	ts.redisDB = &runner.RedisDB{}
	ts.redisProcess = ifrit.Invoke(ts.redisDB)
	Eventually(ts.redisProcess.Ready(), runnerFbk.DefaultStartTimeout).Should(BeClosed())
	Consistently(ts.redisProcess.Wait()).ShouldNot(Receive())
}

func (ts *testSuite) StopRedis() {
	if ts.redisProcess != nil {
		ts.redisProcess.Signal(syscall.SIGTERM)
		Eventually(ts.redisProcess.Wait(), time.Minute).Should(Receive())
	}
}

func (ts *testSuite) StartRobot() {
	robotRunner := ts.networkFound.RobotRunner()
	ts.robotProc = ifrit.Invoke(robotRunner)
	Eventually(ts.robotProc.Ready(), ts.network.EventuallyTimeout).Should(BeClosed())
}

func (ts *testSuite) StopRobot() {
	if ts.robotProc != nil {
		ts.robotProc.Signal(syscall.SIGTERM)
		Eventually(ts.robotProc.Wait(), ts.network.EventuallyTimeout).Should(Receive())
	}
}

func (ts *testSuite) StartChannelTransfer() {
	channelTransferRunner := ts.networkFound.ChannelTransferRunner()
	ts.channelTransferProc = ifrit.Invoke(channelTransferRunner)
	Eventually(ts.channelTransferProc.Ready(), ts.network.EventuallyTimeout).Should(BeClosed())
}

func (ts *testSuite) StopChannelTransfer() {
	if ts.channelTransferProc != nil {
		ts.channelTransferProc.Signal(syscall.SIGTERM)
		Eventually(ts.channelTransferProc.Wait(), ts.network.EventuallyTimeout).Should(Receive())
	}
}

func (ts *testSuite) DeployChaincodes() {
	Expect(ts.options.Channels).NotTo(BeEmpty())
	ts.DeployChaincodesByName(ts.options.Channels)
}

func (ts *testSuite) DeployChaincodesByName(channels []string) {
	for _, channel := range channels {
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
			fabricnetwork.DeployChaincodeFn(ts.components, ts.network, channel, ts.testDir)
		}
	}
}

func (ts *testSuite) DeployFiat(adminAddress, feeSetterAddress, feeAddressSetterAddress string) {
	cmn.DeployFiat(ts.network, ts.components, ts.peer, ts.testDir, ts.skiRobot, adminAddress, feeSetterAddress, feeAddressSetterAddress)
}

func (ts *testSuite) ShutdownNetwork() {
	/*
		if ts.networkProcess != nil {
			ts.networkProcess.Signal(syscall.SIGTERM)
			Eventually(ts.networkProcess.Wait(), ts.network.EventuallyTimeout).Should(Receive())
		}
	*/
	ts.StopPeers()
	ts.StopNetwork()
	ts.StopOrderers()

	err := os.RemoveAll(ts.testDir)
	Expect(err).NotTo(HaveOccurred())
}

func (ts *testSuite) StopPeers() {
	if ts.peerProcess != nil {
		ts.peerProcess.Signal(syscall.SIGTERM)
		Eventually(ts.peerProcess.Wait(), ts.network.EventuallyTimeout).Should(Receive())
	}

	ts.peerProcess = nil
	ts.peerRunner = nil
}

func (ts *testSuite) StopNetwork() {
	if ts.network != nil {
		ts.network.Cleanup()
	}
}

func (ts *testSuite) StopOrderers() {
	for _, ordererInstance := range ts.ordererProcesses {
		ordererInstance.Signal(syscall.SIGTERM)
		Eventually(ordererInstance.Wait(), ts.network.EventuallyTimeout).Should(Receive())
	}

	ts.ordererProcesses = nil
	ts.ordererRunners = nil
}

func (ts *testSuite) ExecuteTask(channel string, chaincode string, method string, args ...string) string {
	task := createTask(method, args...)
	return ts.ExecuteTasks(channel, chaincode, task)
}

func (ts *testSuite) ExecuteTasks(channel string, chaincode string, tasks ...*pbfound.Task) string {
	return executeTasks(
		ts.network,
		ts.peer,
		ts.orderer,
		channel,
		chaincode,
		tasks...,
	).TxID()
}

func (ts *testSuite) ExecuteTaskWithSign(channel string, chaincode string, user *UserFoundation, method string, args ...string) string {
	task, err := CreateTaskWithSignArgs(method, channel, chaincode, user, args...)
	if err != nil {
		panic(err)
	}

	return ts.ExecuteTasks(channel, chaincode, task)
}
