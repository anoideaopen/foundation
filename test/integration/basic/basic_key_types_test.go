package basic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"time"

	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
)

var _ = Describe("Basic foundation tests with different key types", func() {
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

	Describe("foundation test", func() {
		var (
			channels         = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
			ordererRunners   []*ginkgomon.Runner
			redisProcess     ifrit.Process
			redisDB          *runner.RedisDB
			networkFound     *cmn.NetworkFoundation
			robotProc        ifrit.Process
			skiBackend       string
			skiRobot         string
			peer             *nwo.Peer
			admin            *client.UserFoundation
			feeSetter        *client.UserFoundation
			feeAddressSetter *client.UserFoundation
		)
		BeforeEach(func() {
			By("start redis")
			redisDB = &runner.RedisDB{}
			redisProcess = ifrit.Invoke(redisDB)
			Eventually(redisProcess.Ready(), runnerFbk.DefaultStartTimeout).Should(BeClosed())
			Consistently(redisProcess.Wait()).ShouldNot(Receive())
		})
		AfterEach(func() {
			By("stop redis " + redisDB.Address())
			if redisProcess != nil {
				redisProcess.Signal(syscall.SIGTERM)
				Eventually(redisProcess.Wait(), time.Minute).Should(Receive())
			}
		})
		BeforeEach(func() {
			networkConfig := nwo.MultiNodeSmartBFT()
			networkConfig.Channels = nil

			pchs := make([]*nwo.PeerChannel, 0, cap(channels))
			for _, ch := range channels {
				pchs = append(pchs, &nwo.PeerChannel{
					Name:   ch,
					Anchor: true,
				})
			}
			for _, peer := range networkConfig.Peers {
				peer.Channels = pchs
			}

			network = nwo.New(networkConfig, testDir, cli, StartPort(), components)
			cwd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			network.ExternalBuilders = append(network.ExternalBuilders,
				fabricconfig.ExternalBuilder{
					Path:                 filepath.Join(cwd, ".", "externalbuilders", "binary"),
					Name:                 "binary",
					PropagateEnvironment: []string{"GOPROXY"},
				},
			)

			networkFound = cmn.New(network, channels)
			networkFound.Robot.RedisAddresses = []string{redisDB.Address()}

			networkFound.GenerateConfigTree()
			networkFound.Bootstrap()

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

			By("Joining orderers to channels")
			for _, channel := range channels {
				fabricnetwork.JoinChannel(network, channel)
			}

			By("Waiting for followers to see the leader")
			Eventually(ordererRunners[1].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
			Eventually(ordererRunners[2].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
			Eventually(ordererRunners[3].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))

			By("Joining peers to channels")
			for _, channel := range channels {
				network.JoinChannel(channel, network.Orderers[0], network.PeersWithChannel(channel)...)
			}

			peer = network.Peer("Org1", "peer0")

			pathToPrivateKeyBackend := network.PeerUserKey(peer, "User1")
			skiBackend, err = cmn.ReadSKI(pathToPrivateKeyBackend)
			Expect(err).NotTo(HaveOccurred())

			pathToPrivateKeyRobot := network.PeerUserKey(peer, "User2")
			skiRobot, err = cmn.ReadSKI(pathToPrivateKeyRobot)
			Expect(err).NotTo(HaveOccurred())

			admin, err = client.NewUserFoundation(pbfound.KeyType_secp256k1)
			Expect(err).NotTo(HaveOccurred())
			Expect(admin.PrivateKeyBytes).NotTo(Equal(nil))
			feeSetter, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			Expect(feeSetter.PrivateKeyBytes).NotTo(Equal(nil))
			feeAddressSetter, err = client.NewUserFoundation(pbfound.KeyType_secp256k1)
			Expect(err).NotTo(HaveOccurred())
			Expect(feeAddressSetter.PrivateKeyBytes).NotTo(Equal(nil))

			cmn.DeployACL(network, components, peer, testDir, skiBackend, admin.PublicKeyBase58, admin.KeyType)
			cmn.DeployCC(network, components, peer, testDir, skiRobot, admin.AddressBase58Check)
			cmn.DeployFiat(network, components, peer, testDir, skiRobot,
				admin.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check)
			cmn.DeployIndustrial(network, components, peer, testDir, skiRobot,
				admin.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check)
		})
		BeforeEach(func() {
			By("start robot")
			robotRunner := networkFound.RobotRunner()
			robotProc = ifrit.Invoke(robotRunner)
			Eventually(robotProc.Ready(), network.EventuallyTimeout).Should(BeClosed())
		})
		AfterEach(func() {
			By("stop robot")
			if robotProc != nil {
				robotProc.Signal(syscall.SIGTERM)
				Eventually(robotProc.Wait(), network.EventuallyTimeout).Should(Receive())
			}
		})

		It("transfer", func() {
			By("create users")
			user1, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			user2, err := client.NewUserFoundation(pbfound.KeyType_secp256k1)
			Expect(err).NotTo(HaveOccurred())

			By("add users to acl")
			client.AddUser(network, peer, network.Orderers[0], user1)
			client.AddUser(network, peer, network.Orderers[0], user2)

			By("add admin to acl")
			client.AddUser(network, peer, network.Orderers[0], admin)

			By("emit tokens")
			amount := "1"
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, admin,
				"emit", "", client.NewNonceByTime().Get(), nil, user1.AddressBase58Check, amount)

			By("emit check")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
				"balanceOf", user1.AddressBase58Check)

			By("get transfer fee from user1 to user2")
			req := FeeTransferRequestDTO{
				SenderAddress:    user1.AddressBase58Check,
				RecipientAddress: user2.AddressBase58Check,
				Amount:           amount,
			}
			bytes, err := json.Marshal(req)
			Expect(err).NotTo(HaveOccurred())
			fErr := func(out []byte) string {
				Expect(gbytes.BufferWithBytes(out)).To(gbytes.Say("fee address is not set in token config"))
				return ""
			}
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(nil, fErr),
				"getFeeTransfer", string(bytes))

			By("transfer tokens from user1 to user2")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
				client.NewNonceByTime().Get(), nil, user2.AddressBase58Check, amount, "ref transfer")

			By("check balance user1")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
				"balanceOf", user1.AddressBase58Check)

			By("check balance user2")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
				"balanceOf", user2.AddressBase58Check)
		})
	})
})
