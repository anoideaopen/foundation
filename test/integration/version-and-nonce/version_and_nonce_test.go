package version_and_nonce

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

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

var _ = Describe("Version and Nonce Tests", func() {
	var (
		testDir          string
		cli              *docker.Client
		network          *nwo.Network
		networkProcess   ifrit.Process
		ordererProcesses []ifrit.Process
		peerProcesses    ifrit.Process
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

		admin = client.NewUserFoundation()
		Expect(admin.PrivateKey).NotTo(Equal(nil))
		feeSetter = client.NewUserFoundation()
		Expect(feeSetter.PrivateKey).NotTo(Equal(nil))
		feeAddressSetter = client.NewUserFoundation()
		Expect(feeAddressSetter.PrivateKey).NotTo(Equal(nil))

		cmn.DeployACL(network, components, peer, testDir, skiBackend, admin.PublicKeyBase58)
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

	Describe("version tests", func() {
		It("build version", func() {
			f := func(out []byte) string {
				resp := &debug.BuildInfo{}
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Path).To(Equal(cmn.CcModulePath()))

				return ""
			}
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(f, nil), "buildInfo")
		})

		It("core chaincode id name", func() {
			f := func(out []byte) string {
				var resp string
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeEmpty())

				return ""
			}
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(f, nil), "coreChaincodeIDName")
		})

		It("system env", func() {
			f := func(out []byte) string {
				resp := make(map[string]string)
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				_, ok := resp["/etc/issue"]
				Expect(ok).To(BeTrue())

				return ""
			}
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(f, nil), "systemEnv")
		})

		It("embed src files", func() {
			By("get names of files chaincode")
			f := func(out []byte) string {
				var resp []string
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeEmpty())

				return ""
			}
			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				fabricnetwork.CheckResult(f, nil), "nameOfFiles")

			By("get file of chaincode")
			f1 := func(out []byte) string {
				var resp string
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeEmpty())
				Expect(resp[8:23]).To(Equal("industrialtoken"))

				return ""
			}
			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				fabricnetwork.CheckResult(f1, nil), "srcFile", "industrial_token/token.go")

			By("get part file of chaincode")
			f2 := func(out []byte) string {
				var resp string
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeEmpty())
				Expect(resp).To(Equal("industrialtoken"))

				return ""
			}
			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				fabricnetwork.CheckResult(f2, nil), "srcPartFile", "industrial_token/token.go", "8", "23")
		})
	})

	It("nonce test", func() {
		By("add admin to acl")
		client.AddUser(network, peer, network.Orderers[0], admin)

		By("add user to acl")
		user1 := client.NewUserFoundation()
		client.AddUser(network, peer, network.Orderers[0], user1)

		By("prepare nonces")
		nonce := client.NewNonceByTime()
		nonce1 := nonce.Get()
		nonce.Add(51000)
		nonce2 := nonce.Get()
		nonce.Next()
		nonce3 := nonce.Get()
		nonce.Next()
		nonce4 := nonce.Get()

		emitAmount := "1"

		By("emit tokens 1")
		client.TxInvokeWithSign(network, peer, network.Orderers[0],
			cmn.ChannelFiat, cmn.ChannelFiat, admin,
			"emit", "", nonce3, user1.AddressBase58Check, emitAmount)

		By("emit tokens 2")
		client.TxInvokeWithSign(network, peer, network.Orderers[0],
			cmn.ChannelFiat, cmn.ChannelFiat, admin,
			"emit", "", nonce2, user1.AddressBase58Check, emitAmount)

		By("emit tokens 3")
		client.TxInvokeWithSign(network, peer, network.Orderers[0],
			cmn.ChannelFiat, cmn.ChannelFiat, admin,
			"emit", "", nonce1, user1.AddressBase58Check, emitAmount)

		By("emit tokens 4")
		client.TxInvokeWithSign(network, peer, network.Orderers[0],
			cmn.ChannelFiat, cmn.ChannelFiat, admin,
			"emit", "", nonce3, user1.AddressBase58Check, emitAmount)

		By("emit tokens 5")
		client.TxInvokeWithSign(network, peer, network.Orderers[0],
			cmn.ChannelFiat, cmn.ChannelFiat, admin,
			"emit", "", nonce4, user1.AddressBase58Check, emitAmount)

		By("emit check")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance("3"), nil),
			"balanceOf", user1.AddressBase58Check)
	})
})
