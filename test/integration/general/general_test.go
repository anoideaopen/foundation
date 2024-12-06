package general

import (
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/chaincode/fiat/service"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
	docker "github.com/fsouza/go-dockerclient"
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
)

var _ = Describe("Basic foundation Tests", func() {

	Describe("smartbft standart test", func() {
		var (
			testDir          string
			cli              *docker.Client
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
			testDir, err = os.MkdirTemp("", "foundation")
			Expect(err).NotTo(HaveOccurred())

			cli, err = docker.NewClientFromEnv()
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
			err := os.RemoveAll(testDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("smartbft multiple nodes stop start all nodes", func() {
			networkConfig := nwo.MultiNodeSmartBFT()
			networkConfig.Channels = nil
			channel := "testchannel1"

			network = nwo.New(networkConfig, testDir, cli, StartPort(), components)
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

			peerGroupRunner, _ := fabricnetwork.PeerGroupRunners(network)
			peerProcesses = ifrit.Invoke(peerGroupRunner)
			Eventually(peerProcesses.Ready(), network.EventuallyTimeout).Should(BeClosed())
			peer := network.Peer("Org1", "peer0")

			fabricnetwork.JoinChannel(network, channel)

			By("Waiting for followers to see the leader")
			Eventually(ordererRunners[1].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
			Eventually(ordererRunners[2].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
			Eventually(ordererRunners[3].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))

			By("Joining peers to testchannel1")
			network.JoinChannel(channel, network.Orderers[0], network.PeersWithChannel(channel)...)

			By("Deploying chaincode")
			fabricnetwork.DeployChaincodeFn(components, network, channel, testDir)

			By("querying the chaincode")
			sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
				ChannelID: channel,
				Name:      "mycc",
				Ctor:      cmn.CtorFromSlice([]string{"query", "a"}),
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
			Expect(sess).To(gbytes.Say("100"))

			By("invoking the chaincode")
			fabricnetwork.InvokeQuery(network, peer, network.Orderers[1], channel, 90)

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
			fabricnetwork.InvokeQuery(network, peer, network.Orderers[2], channel, 80)
		})
	})

	Describe("foundation test", func() {
		var (
			channels = []string{cmn.ChannelACL, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
			ts       client.TestSuite
		)

		BeforeEach(func() {
			ts = client.NewTestSuite(components)
		})
		AfterEach(func() {
			ts.ShutdownNetwork()
		})

		BeforeEach(func() {
			By("start redis")
			ts.StartRedis()
		})
		BeforeEach(func() {
			ts.InitNetwork(channels, integration.ConfigBasePort)
			ts.DeployChaincodes()
		})
		BeforeEach(func() {
			By("start robot")
			ts.StartRobot()
		})
		AfterEach(func() {
			By("stop robot")
			ts.StopRobot()
			By("stop redis")
			ts.StopRedis()
		})

		It("example test", func() {
			By("add admin to acl")
			ts.AddAdminToACL()

			By("add user to acl")
			user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			ts.AddUser(user1)

			By("emit tokens")
			emitAmount := "1"
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, emitAmount).CheckErrorIsNil()

			By("emit check")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(emitAmount)

			By("add balance by admin to user1 (gRPC router)")
			req := &service.BalanceAdjustmentRequest{
				Address: &service.Address{
					Base58Check: user1.AddressBase58Check,
				},
				Amount: &service.BigInt{
					Value: emitAmount,
				},
				Reason: "some important reason",
			}
			rawReq, _ := protojson.Marshal(req)

			By("add balance by admin to user1 gRPC router")
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"/foundationtoken.FiatService/AddBalanceByAdmin", "", client.NewNonceByTime().Get(), string(rawReq)).CheckErrorIsNil()

			newBalance := "2"
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(newBalance)
		})
	})
})
