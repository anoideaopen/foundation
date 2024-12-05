package client

import (
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anoideaopen/foundation/mocks"
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

type FoundationTestSuite struct {
	Network       *nwo.Network
	NetworkFound  *cmn.NetworkFoundation
	Peer          *nwo.Peer
	Orderer       *nwo.Orderer
	Org1Name      string
	Org2Name      string
	MainUserName  string
	RobotUserName string

	components          *nwo.Components
	options             *networkOptions
	userOptions         *userOptions
	testDir             string
	dockerClient        *docker.Client
	redisDB             *runner.RedisDB
	redisProcess        ifrit.Process
	robotProc           ifrit.Process
	ordererProcesses    []ifrit.Process
	peerProcess         ifrit.Process
	channelTransferProc ifrit.Process
	peerRunner          ifrit.Runner
	ordererRunners      []*ginkgomon.Runner
	admin               *mocks.UserFoundation
	feeSetter           *mocks.UserFoundation
	feeAddressSetter    *mocks.UserFoundation
	skiBackend          string
	skiRobot            string

	isInit bool
}

func NewTestSuite(components *nwo.Components, opts ...UserOption) *FoundationTestSuite {
	testDir, err := os.MkdirTemp("", "foundation")
	Expect(err).NotTo(HaveOccurred())

	dockerClient, err := docker.NewClientFromEnv()
	Expect(err).NotTo(HaveOccurred())

	ts := &FoundationTestSuite{
		Org1Name:      defaultOrg1Name,
		Org2Name:      defaultOrg2Name,
		MainUserName:  defaultMainUserName,
		RobotUserName: defaultRobotUserName,
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
		userOptions: &userOptions{
			AdminKeyType:            pbfound.KeyType_ed25519,
			FeeSetterKeyType:        pbfound.KeyType_ed25519,
			FeeAddressSetterKeyType: pbfound.KeyType_ed25519,
		},
		isInit: false,
	}

	for _, opt := range opts {
		err := opt(ts.userOptions)
		Expect(err).NotTo(HaveOccurred())
	}

	return ts
}

func (ts *FoundationTestSuite) InitNetwork(channels []string, testPort integration.TestPortRange, opts ...NetworkOption) {
	ts.options.Channels = make([]*cmn.Channel, len(channels))
	for i, channel := range channels {
		ts.options.Channels[i] = &cmn.Channel{Name: channel}
	}
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
			Name:   ch.Name,
			Anchor: true,
		})
	}
	for _, peer := range networkConfig.Peers {
		peer.Channels = peerChannels
	}

	ts.Network = nwo.New(networkConfig, ts.testDir, ts.dockerClient, ts.options.TestPort.StartPortForNode(), ts.components)

	cwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	ts.Network.ExternalBuilders = append(ts.Network.ExternalBuilders,
		fabricconfig.ExternalBuilder{
			Path:                 filepath.Join(cwd, ".", "externalbuilders", "binary"),
			Name:                 "binary",
			PropagateEnvironment: []string{"GOPROXY"},
		},
	)

	ts.NetworkFound = cmn.New(
		ts.Network,
		ts.options.Channels,
		cmn.WithRobotCfg(ts.options.RobotCfg),
		cmn.WithChannelTransferCfg(ts.options.ChannelTransferCfg),
		cmn.WithRobotTemplate(ts.options.Templates.Robot),
		cmn.WithChannelTransferTemplate(ts.options.Templates.ChannelTransfer),
	)

	if ts.redisDB != nil {
		ts.NetworkFound.Robot.RedisAddresses = []string{ts.redisDB.Address()}
		ts.NetworkFound.ChannelTransfer.RedisAddresses = []string{ts.redisDB.Address()}
	}

	ts.NetworkFound.GenerateConfigTree()
	ts.NetworkFound.Bootstrap()

	for _, orderer := range ts.Network.Orderers {
		ordererRunner := ts.Network.OrdererRunner(orderer)
		ordererRunner.Command.Env = append(ordererRunner.Command.Env, "FABRIC_LOGGING_SPEC=orderer.consensus.smartbft=debug:grpc=debug")
		ts.ordererRunners = append(ts.ordererRunners, ordererRunner)
		proc := ifrit.Invoke(ordererRunner)
		ts.ordererProcesses = append(ts.ordererProcesses, proc)
		Eventually(proc.Ready(), ts.Network.EventuallyTimeout).Should(BeClosed())
	}

	peerGroupRunner, _ := fabricnetwork.PeerGroupRunners(ts.Network)
	ts.peerProcess = ifrit.Invoke(peerGroupRunner)
	Eventually(ts.peerProcess.Ready(), ts.Network.EventuallyTimeout).Should(BeClosed())

	ts.Peer = ts.Network.Peer(ts.Org1Name, defaultPeerName)
	ts.Orderer = ts.Network.Orderers[0]

	By("Joining orderers to channels")
	for _, channel := range ts.options.Channels {
		fabricnetwork.JoinChannel(ts.Network, channel.Name)
	}

	By("Waiting for followers to see the leader")
	Eventually(ts.ordererRunners[1].Err(), ts.Network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
	Eventually(ts.ordererRunners[2].Err(), ts.Network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
	Eventually(ts.ordererRunners[3].Err(), ts.Network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))

	By("Joining peers to channels")
	for _, channel := range ts.options.Channels {
		ts.Network.JoinChannel(channel.Name, ts.Orderer, ts.Network.PeersWithChannel(channel.Name)...)
	}

	pathToPrivateKeyBackend := ts.Network.PeerUserKey(ts.Peer, ts.MainUserName)
	skiBackend, err := cmn.ReadSKI(pathToPrivateKeyBackend)
	Expect(err).NotTo(HaveOccurred())

	pathToPrivateKeyRobot := ts.Network.PeerUserKey(ts.Peer, ts.RobotUserName)
	skiRobot, err := cmn.ReadSKI(pathToPrivateKeyRobot)
	Expect(err).NotTo(HaveOccurred())

	ts.skiBackend = skiBackend
	ts.skiRobot = skiRobot

	ts.admin, err = mocks.NewUserFoundation(ts.userOptions.AdminKeyType)
	Expect(err).NotTo(HaveOccurred())
	Expect(ts.admin.PrivateKeyBytes).NotTo(Equal(nil))

	ts.feeSetter, err = mocks.NewUserFoundation(ts.userOptions.FeeSetterKeyType)
	Expect(err).NotTo(HaveOccurred())
	Expect(ts.feeSetter.PrivateKeyBytes).NotTo(Equal(nil))

	ts.feeAddressSetter, err = mocks.NewUserFoundation(ts.userOptions.FeeAddressSetterKeyType)
	Expect(err).NotTo(HaveOccurred())
	Expect(ts.feeAddressSetter.PrivateKeyBytes).NotTo(Equal(nil))

	ts.isInit = true
}

func (ts *FoundationTestSuite) Admin() *mocks.UserFoundation {
	return ts.admin
}

func (ts *FoundationTestSuite) FeeSetter() *mocks.UserFoundation {
	return ts.feeSetter
}

func (ts *FoundationTestSuite) FeeAddressSetter() *mocks.UserFoundation {
	return ts.feeAddressSetter
}

/*
func (ts *FoundationTestSuite) Network() *nwo.Network {
	return ts.network
}

func (ts *FoundationTestSuite) NetworkFound() *cmn.NetworkFoundation {
	return ts.networkFound
}

func (ts *FoundationTestSuite) Peer() *nwo.Peer {
	return ts.peer
}
*/

func (ts *FoundationTestSuite) TestDir() string {
	return ts.testDir
}

func (ts *FoundationTestSuite) DockerClient() *docker.Client {
	return ts.dockerClient
}

func (ts *FoundationTestSuite) CiData(opts ...CiDataOption) ntesting.CiTestData {
	Expect(ts.isInit).To(BeTrue())
	Expect(ts.redisDB).NotTo(BeNil())

	// setting default values
	ciData := ntesting.CiTestData{
		RedisAddr:             ts.redisDB.Address(),
		RedisPass:             "",
		HlfProfilePath:        ts.NetworkFound.ConnectionPath(ts.MainUserName),
		HlfFiatChannel:        cmn.ChannelFiat,
		HlfCcChannel:          cmn.ChannelCC,
		HlfIndustrialChannel:  cmn.ChannelIndustrial,
		HlfNoCcChannel:        "",
		HlfUserName:           "backend",
		HlfCert:               ts.Network.PeerUserKey(ts.Peer, ts.MainUserName),
		HlfFiatOwnerKey:       ts.admin.PublicKeyBase58,
		HlfCcOwnerKey:         ts.admin.PublicKeyBase58,
		HlfIndustrialOwnerKey: ts.admin.PublicKeyBase58,
		HlfIndustrialGroup1:   "",
		HlfIndustrialGroup2:   "",
		HlfSk:                 ts.Network.PeerUserKey(ts.Peer, ts.MainUserName),
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

func (ts *FoundationTestSuite) StartRedis() {
	ts.redisDB = &runner.RedisDB{}
	ts.redisProcess = ifrit.Invoke(ts.redisDB)
	Eventually(ts.redisProcess.Ready(), runnerFbk.DefaultStartTimeout).Should(BeClosed())
	Consistently(ts.redisProcess.Wait()).ShouldNot(Receive())
}

func (ts *FoundationTestSuite) StopRedis() {
	if ts.redisProcess != nil {
		ts.redisProcess.Signal(syscall.SIGTERM)
		Eventually(ts.redisProcess.Wait(), time.Minute).Should(Receive())
	}
}

func (ts *FoundationTestSuite) StartRobot() {
	robotRunner := ts.NetworkFound.RobotRunner()
	ts.robotProc = ifrit.Invoke(robotRunner)
	Eventually(ts.robotProc.Ready(), ts.Network.EventuallyTimeout).Should(BeClosed())
}

func (ts *FoundationTestSuite) StopRobot() {
	if ts.robotProc != nil {
		ts.robotProc.Signal(syscall.SIGTERM)
		Eventually(ts.robotProc.Wait(), ts.Network.EventuallyTimeout).Should(Receive())
	}
}

func (ts *FoundationTestSuite) StartChannelTransfer() {
	channelTransferRunner := ts.NetworkFound.ChannelTransferRunner()
	ts.channelTransferProc = ifrit.Invoke(channelTransferRunner)
	Eventually(ts.channelTransferProc.Ready(), ts.Network.EventuallyTimeout).Should(BeClosed())
}

func (ts *FoundationTestSuite) StopChannelTransfer() {
	if ts.channelTransferProc != nil {
		ts.channelTransferProc.Signal(syscall.SIGTERM)
		Eventually(ts.channelTransferProc.Wait(), ts.Network.EventuallyTimeout).Should(Receive())
	}
}

func (ts *FoundationTestSuite) DeployChaincodes() {
	Expect(ts.options.Channels).NotTo(BeEmpty())
	channelNames := make([]string, len(ts.options.Channels))
	for i, ch := range ts.options.Channels {
		channelNames[i] = ch.Name
	}
	ts.DeployChaincodesByName(channelNames)
}

func (ts *FoundationTestSuite) DeployChaincodesByName(channels []string) {
	for _, channel := range channels {
		switch channel {
		case cmn.ChannelAcl:
			cmn.DeployACL(ts.Network, ts.components, ts.Peer, ts.testDir, ts.skiBackend, ts.admin.PublicKeyBase58, ts.admin.KeyType)
		case cmn.ChannelFiat:
			cmn.DeployFiat(ts.Network, ts.components, ts.Peer, ts.testDir, ts.skiRobot, ts.admin.AddressBase58Check, ts.feeSetter.AddressBase58Check, ts.feeAddressSetter.AddressBase58Check)
		case cmn.ChannelCC:
			cmn.DeployCC(ts.Network, ts.components, ts.Peer, ts.testDir, ts.skiRobot, ts.admin.AddressBase58Check)
		case cmn.ChannelIndustrial:
			cmn.DeployIndustrial(ts.Network, ts.components, ts.Peer, ts.testDir, ts.skiRobot, ts.admin.AddressBase58Check, ts.feeSetter.AddressBase58Check, ts.feeAddressSetter.AddressBase58Check)
		default:
			fabricnetwork.DeployChaincodeFn(ts.components, ts.Network, channel, ts.testDir)
		}
	}
}

func (ts *FoundationTestSuite) DeployFiat(adminAddress, feeSetterAddress, feeAddressSetterAddress string) {
	cmn.DeployFiat(ts.Network, ts.components, ts.Peer, ts.testDir, ts.skiRobot, adminAddress, feeSetterAddress, feeAddressSetterAddress)
}

func (ts *FoundationTestSuite) ShutdownNetwork() {
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

func (ts *FoundationTestSuite) StopPeers() {
	if ts.peerProcess != nil {
		ts.peerProcess.Signal(syscall.SIGTERM)
		Eventually(ts.peerProcess.Wait(), ts.Network.EventuallyTimeout).Should(Receive())
	}

	ts.peerProcess = nil
	ts.peerRunner = nil
}

func (ts *FoundationTestSuite) StopNetwork() {
	if ts.Network != nil {
		ts.Network.Cleanup()
	}
}

func (ts *FoundationTestSuite) StopOrderers() {
	for _, ordererInstance := range ts.ordererProcesses {
		ordererInstance.Signal(syscall.SIGTERM)
		Eventually(ordererInstance.Wait(), ts.Network.EventuallyTimeout).Should(Receive())
	}

	ts.ordererProcesses = nil
	ts.ordererRunners = nil
}
