package client

import (
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
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
	"os"
	"path/filepath"
	"syscall"
	"time"
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
	channels            []string
	network             *nwo.Network
	networkFound        *cmn.NetworkFoundation
	peer                *nwo.Peer
	orderer             *nwo.Orderer
	redisProcess        ifrit.Process
	redisDB             *runner.RedisDB
	robotProc           ifrit.Process
	networkProcess      ifrit.Process
	ordererProcesses    []ifrit.Process
	peerProcess         ifrit.Process
	channelTransferProc ifrit.Process
	testDir             string
	dockerClient        *docker.Client
	org1Name            string
	org2Name            string
	mainUserName        string
	robotUserName       string
	admin               *UserFoundation
	feeSetter           *UserFoundation
	feeAddressSetter    *UserFoundation
	skiBackend          string
	skiRobot            string
}

func initPeer(network *nwo.Network, orgName string) *nwo.Peer {
	return network.Peer(orgName, defaultPeerName)
}

func startPort(portRange integration.TestPortRange) int {
	return portRange.StartPortForNode()
}

func (ts *testSuite) InitNetwork(testPort integration.TestPortRange) {
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

	ts.network = nwo.New(networkConfig, ts.testDir, ts.dockerClient, startPort(testPort), ts.components)

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
	ts.networkFound.Robot.RedisAddresses = []string{ts.redisDB.Address()}

	ts.networkFound.GenerateConfigTree()
	ts.networkFound.Bootstrap()

	for _, orderer := range ts.network.Orderers {
		ordererRunner := ts.network.OrdererRunner(orderer)
		ordererRunner.Command.Env = append(ordererRunner.Command.Env, "FABRIC_LOGGING_SPEC=orderer.consensus.smartbft=debug:grpc=debug")
		ordererRunners = append(ordererRunners, ordererRunner)
		proc := ifrit.Invoke(ordererRunner)
		ts.ordererProcesses = append(ts.ordererProcesses, proc)
		Eventually(proc.Ready(), ts.network.EventuallyTimeout).Should(BeClosed())
	}

	peerGroupRunner, _ := fabricnetwork.PeerGroupRunners(ts.network)
	ts.peerProcess = ifrit.Invoke(peerGroupRunner)
	Eventually(ts.peerProcess.Ready(), ts.network.EventuallyTimeout).Should(BeClosed())

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

	ts.admin, err = NewUserFoundation(pbfound.KeyType_ed25519)
	Expect(err).NotTo(HaveOccurred())
	Expect(ts.admin.PrivateKeyBytes).NotTo(Equal(nil))

	ts.feeSetter, err = NewUserFoundation(pbfound.KeyType_ed25519)
	Expect(err).NotTo(HaveOccurred())
	Expect(ts.feeSetter.PrivateKeyBytes).NotTo(Equal(nil))

	ts.feeAddressSetter, err = NewUserFoundation(pbfound.KeyType_ed25519)
	Expect(err).NotTo(HaveOccurred())
	Expect(ts.feeAddressSetter.PrivateKeyBytes).NotTo(Equal(nil))
}

func NewTestSuite(
	components *nwo.Components,
	channels []string,
) TestSuite {
	testDir, err := os.MkdirTemp("", "foundation")
	Expect(err).NotTo(HaveOccurred())

	dockerClient, err := docker.NewClientFromEnv()
	Expect(err).NotTo(HaveOccurred())

	ts := &testSuite{
		org1Name:         defaultOrg1Name,
		org2Name:         defaultOrg2Name,
		mainUserName:     defaultMainUserName,
		robotUserName:    defaultRobotUserName,
		channels:         channels,
		components:       components,
		testDir:          testDir,
		dockerClient:     dockerClient,
		networkProcess:   nil,
		ordererProcesses: nil,
		peerProcess:      nil,
	}

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

func (ts *testSuite) OrdererProcesses() []ifrit.Process {
	return ts.ordererProcesses
}

func (ts *testSuite) PeerProcess() ifrit.Process {
	return ts.peerProcess
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

func (ts *testSuite) ShutdownNetwork() {
	if ts.networkProcess != nil {
		ts.networkProcess.Signal(syscall.SIGTERM)
		Eventually(ts.networkProcess.Wait(), ts.network.EventuallyTimeout).Should(Receive())
	}
	if ts.peerProcess != nil {
		ts.peerProcess.Signal(syscall.SIGTERM)
		Eventually(ts.peerProcess.Wait(), ts.network.EventuallyTimeout).Should(Receive())
	}
	if ts.network != nil {
		ts.network.Cleanup()
	}
	for _, ordererInstance := range ts.OrdererProcesses() {
		ordererInstance.Signal(syscall.SIGTERM)
		Eventually(ordererInstance.Wait(), ts.network.EventuallyTimeout).Should(Receive())
	}
	err := os.RemoveAll(ts.testDir)
	Expect(err).NotTo(HaveOccurred())
}
