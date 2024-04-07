package integration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/integration/channelparticipation"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
	"github.com/tedsuo/ifrit/grouper"
)

var _ = Describe("Foundation Tests", func() {
	var (
		testDir          string
		client           *docker.Client
		network          *nwo.Network
		networkProcess   ifrit.Process
		ordererProcesses []ifrit.Process
		peerProcesses    ifrit.Process
	)

	BeforeEach(func() {
		networkProcess = nil
		ordererProcesses = nil
		peerProcesses = nil
		var err error
		testDir, err = os.MkdirTemp("", "foundation-test")
		Expect(err).NotTo(HaveOccurred())

		client, err = docker.NewClientFromEnv()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if networkProcess != nil {
			networkProcess.Signal(syscall.SIGTERM)
			Eventually(networkProcess.Wait(), network.EventuallyTimeout).Should(Receive())
		}
		if peerProcesses != nil {
			peerProcesses.Signal(syscall.SIGTERM)
			Eventually(peerProcesses.Wait(), network.EventuallyTimeout).Should(Receive())
		}
		if network != nil {
			network.Cleanup()
		}
		for _, ordererInstance := range ordererProcesses {
			ordererInstance.Signal(syscall.SIGTERM)
			Eventually(ordererInstance.Wait(), network.EventuallyTimeout).Should(Receive())
		}
		os.RemoveAll(testDir)
	})

	Describe("foundation work", func() {
		It("smartbft multiple nodes stop start all nodes", func() {
			networkConfig := nwo.MultiNodeSmartBFT()
			networkConfig.Channels = nil
			channel := "testchannel1"

			network = nwo.New(networkConfig, testDir, client, StartPort(), components)
			cwd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			network.ExternalBuilders = append(network.ExternalBuilders,
				fabricconfig.ExternalBuilder{
					Path:                 filepath.Join(cwd, ".", "externalbuilders", "binary"),
					Name:                 "binary",
					PropagateEnvironment: []string{"GOPROXY"},
				})

			network.GenerateConfigTree()
			network.Bootstrap()

			var ordererRunners []*ginkgomon.Runner
			for _, orderer := range network.Orderers {
				runner := network.OrdererRunner(orderer)
				runner.Command.Env = append(runner.Command.Env, "FABRIC_LOGGING_SPEC=orderer.consensus.smartbft=debug:grpc=debug")
				ordererRunners = append(ordererRunners, runner)
				proc := ifrit.Invoke(runner)
				ordererProcesses = append(ordererProcesses, proc)
				Eventually(proc.Ready(), network.EventuallyTimeout).Should(BeClosed())
			}

			peerGroupRunner, _ := peerGroupRunners(network)
			peerProcesses = ifrit.Invoke(peerGroupRunner)
			Eventually(peerProcesses.Ready(), network.EventuallyTimeout).Should(BeClosed())
			peer := network.Peer("Org1", "peer0")

			joinChannel(network, channel)

			By("Waiting for followers to see the leader")
			Eventually(ordererRunners[1].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
			Eventually(ordererRunners[2].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
			Eventually(ordererRunners[3].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))

			By("Joining peers to testchannel1")
			network.JoinChannel(channel, network.Orderers[0], network.PeersWithChannel(channel)...)

			By("Deploying chaincode")
			deployChaincode(network, channel, testDir)

			By("querying the chaincode")
			sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
				ChannelID: channel,
				Name:      "mycc",
				Ctor:      `{"Args":["query","a"]}`,
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
			Expect(sess).To(gbytes.Say("100"))

			By("invoking the chaincode")
			invokeQuery(network, peer, network.Orderers[1], channel, 90)

			By("Taking down all the orderers")
			for _, proc := range ordererProcesses {
				proc.Signal(syscall.SIGTERM)
				Eventually(proc.Wait(), network.EventuallyTimeout).Should(Receive())
			}

			ordererRunners = nil
			ordererProcesses = nil
			By("Bringing up all the nodes")
			for _, orderer := range network.Orderers {
				runner := network.OrdererRunner(orderer)
				runner.Command.Env = append(runner.Command.Env, "FABRIC_LOGGING_SPEC=orderer.consensus.smartbft=debug:grpc=debug")
				ordererRunners = append(ordererRunners, runner)
				proc := ifrit.Invoke(runner)
				ordererProcesses = append(ordererProcesses, proc)
				Eventually(proc.Ready(), network.EventuallyTimeout).Should(BeClosed())
			}

			By("Waiting for followers to see the leader, again")
			Eventually(ordererRunners[1].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1 channel=testchannel1"))
			Eventually(ordererRunners[2].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1 channel=testchannel1"))
			Eventually(ordererRunners[3].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1 channel=testchannel1"))

			By("invoking the chaincode, again")
			invokeQuery(network, peer, network.Orderers[2], channel, 80)
		})
	})
})

func peerGroupRunners(n *nwo.Network) (ifrit.Runner, []*ginkgomon.Runner) {
	var runners []*ginkgomon.Runner
	members := grouper.Members{}
	for _, p := range n.Peers {
		runner := n.PeerRunner(p)
		members = append(members, grouper.Member{Name: p.ID(), Runner: runner})
		runners = append(runners, runner)
	}
	return grouper.NewParallel(syscall.SIGTERM, members), runners
}

func joinChannel(network *nwo.Network, channel string, onlyNodes ...int) {
	genesisBlockBytes, err := os.ReadFile(network.OutputBlockPath(channel))
	if err != nil && errors.Is(err, syscall.ENOENT) {
		sess, err := network.ConfigTxGen(commands.OutputBlock{
			ChannelID:   channel,
			Profile:     network.Profiles[0].Name,
			ConfigPath:  network.RootDir,
			OutputBlock: network.OutputBlockPath(channel),
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))

		genesisBlockBytes, err = os.ReadFile(network.OutputBlockPath(channel))
		Expect(err).NotTo(HaveOccurred())
	}

	genesisBlock := &common.Block{}
	err = proto.Unmarshal(genesisBlockBytes, genesisBlock)
	Expect(err).NotTo(HaveOccurred())

	expectedChannelInfoPT := channelparticipation.ChannelInfo{
		Name:              channel,
		URL:               "/participation/v1/channels/" + channel,
		Status:            "active",
		ConsensusRelation: "consenter",
		Height:            1,
	}

	if len(onlyNodes) != 0 {
		for _, i := range onlyNodes {
			o := network.Orderers[i]
			By("joining " + o.Name + " to channel as a consenter")
			channelparticipation.Join(network, o, channel, genesisBlock, expectedChannelInfoPT)
			channelInfo := channelparticipation.ListOne(network, o, channel)
			Expect(channelInfo).To(Equal(expectedChannelInfoPT))
		}

		return
	}

	for _, o := range network.Orderers {
		By("joining " + o.Name + " to channel as a consenter")
		channelparticipation.Join(network, o, channel, genesisBlock, expectedChannelInfoPT)
		channelInfo := channelparticipation.ListOne(network, o, channel)
		Expect(channelInfo).To(Equal(expectedChannelInfoPT))
	}
}

func deployChaincode(network *nwo.Network, channel string, testDir string) {
	nwo.DeployChaincode(network, channel, network.Orderers[0], nwo.Chaincode{
		Name:            "mycc",
		Version:         "0.0",
		Path:            components.Build("github.com/hyperledger/fabric/integration/chaincode/simple/cmd"),
		Lang:            "binary",
		PackageFile:     filepath.Join(testDir, "simplecc.tar.gz"),
		Ctor:            `{"Args":["init","a","100","b","200"]}`,
		SignaturePolicy: `AND ('Org1MSP.member','Org2MSP.member')`,
		Sequence:        "1",
		InitRequired:    true,
		Label:           "my_prebuilt_chaincode",
	})
}

func invokeQuery(network *nwo.Network, peer *nwo.Peer, orderer *nwo.Orderer, channel string, expectedBalance int) {
	sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
		ChannelID: channel,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      "mycc",
		Ctor:      `{"Args":["invoke","a","b","10"]}`,
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	queryExpect(network, peer, channel, "a", expectedBalance)
}

func queryExpect(network *nwo.Network, peer *nwo.Peer, channel string, key string, expectedBalance int) {
	Eventually(func() string {
		sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
			ChannelID: channel,
			Name:      "mycc",
			Ctor:      fmt.Sprintf(`{"Args":["query","%s"]}`, key),
		})
		Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit())
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		outStr := strings.TrimSpace(string(sess.Out.Contents()))
		if outStr != fmt.Sprintf("%d", expectedBalance) {
			return fmt.Sprintf("Error: expected: %d, received %s", expectedBalance, outStr)
		}
		return ""
	}, network.EventuallyTimeout, time.Second).Should(BeEmpty())
}
