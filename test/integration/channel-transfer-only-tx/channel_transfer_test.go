package channel_transfer_only_tx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types/big"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
)

var _ = Describe("Channel transfer only tx foundation Tests", func() {
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

		admin, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())
		Expect(admin.PrivateKeyBytes).NotTo(Equal(nil))

		feeSetter, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())
		Expect(feeSetter.PrivateKeyBytes).NotTo(Equal(nil))

		feeAddressSetter, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
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

	AfterEach(func() {
		By("stop redis " + redisDB.Address())
		if redisProcess != nil {
			redisProcess.Signal(syscall.SIGTERM)
			Eventually(redisProcess.Wait(), time.Minute).Should(Receive())
		}
	})

	Describe("channel transfer test", func() {
		var (
			user1                *client.UserFoundation
			transferAmount       = "450"
			balanceAfterTransfer = "550"
			emitAmount           = "1000"
			id                   string
			id2                  string
			transferItems        []core.TransferItem
		)

		BeforeEach(func() {
			By("add admin to acl")
			client.AddUser(network, peer, network.Orderers[0], admin)

			By("add user to acl")
			var err error
			user1, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			client.AddUser(network, peer, network.Orderers[0], user1)

			id = uuid.NewString()
			id2 = uuid.NewString()

			By("emit tokens 1000")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, admin,
				"emit", "", client.NewNonceByTime().Get(), nil, user1.AddressBase58Check, emitAmount)

			By("emit check")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
				"balanceOf", user1.AddressBase58Check)

			By("initialize industrial")
			client.NBTxInvokeWithSign(network, peer, network.Orderers[0], nil, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				admin, "initialize", "", client.NewNonceByTime().Get())

			transferItems = []core.TransferItem{
				{
					Token:  "INDUSTRIAL_202009",
					Amount: big.NewInt(10000000000000),
				},
				{
					Token:  "INDUSTRIAL_202010",
					Amount: big.NewInt(100000000000000),
				},
				{
					Token:  "INDUSTRIAL_202011",
					Amount: big.NewInt(200000000000000),
				},
				{
					Token:  "INDUSTRIAL_202012",
					Amount: big.NewInt(50000000000000),
				},
			}

			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, item.Amount.String()), nil),
					"industrialBalanceOf", admin.AddressBase58Check)
				client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					admin, "transferIndustrial", "", client.NewNonceByTime().Get(), nil,
					user1.AddressBase58Check, group, item.Amount.String(), "comment")
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, item.Amount.String()), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}
		})

		It("by customer success", func() {
			By("FORWARD")

			By("channel transfer by customer forward")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id, "CC", "FIAT", transferAmount)

			By("check balance after transfer")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(balanceAfterTransfer), nil),
				"balanceOf", user1.AddressBase58Check)

			By("get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)
				return ""
			}
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
				"channelTransferFrom", id)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			client.TxInvokeByRobot(network, peer, network.Orderers[0],
				cmn.ChannelCC, cmn.ChannelCC, nil, "createCCTransferTo", from)

			By("check allowed balance 1")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(transferAmount), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
				"channelTransferTo", id)

			By("commit cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat, "commitCCTransferFrom", id)

			By("delete cc transfer to")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferTo", id)

			By("delete cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferFrom", id)

			By("check allowed balance 2")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(transferAmount), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("check fiat balance")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(balanceAfterTransfer), nil),
				"balanceOf", user1.AddressBase58Check)

			By("BACKWARD")

			By("channel transfer by customer backward")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelCC, cmn.ChannelCC, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id2, "FIAT", "FIAT", transferAmount)

			By("check allowed balance after transfer")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("get channel transfer from")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrFrom, nil),
				"channelTransferFrom", id2)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			client.TxInvokeByRobot(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, nil, "createCCTransferTo", from)

			By("check fiat balance 1")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
				"balanceOf", user1.AddressBase58Check)

			By("channel transfer to")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrTo, nil),
				"channelTransferTo", id2)

			By("commit cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2)

			By("delete cc transfer to")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferTo", id2)

			By("delete cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2)

			By("check allowed balance")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("check fiat balance 2")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
				"balanceOf", user1.AddressBase58Check)
		})

		It("channel transfer by admin success", func() {
			By("FORWARD")

			By("channel transfer forward")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, admin, "channelTransferByAdmin", "",
				client.NewNonceByTime().Get(), nil, id, "CC", user1.AddressBase58Check, "FIAT", transferAmount)

			By("check balance after transfer")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(balanceAfterTransfer), nil),
				"balanceOf", user1.AddressBase58Check)

			By("get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)

				return ""
			}
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
				"channelTransferFrom", id)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			client.TxInvokeByRobot(network, peer, network.Orderers[0],
				cmn.ChannelCC, cmn.ChannelCC, nil, "createCCTransferTo", from)

			By("check allowed balance 1")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(transferAmount), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
				"channelTransferTo", id)

			By("commit cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat, "commitCCTransferFrom", id)

			By("delete cc transfer to")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferTo", id)

			By("delete cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferFrom", id)

			By("check allowed balance 2")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(transferAmount), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("check fiat balance")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(balanceAfterTransfer), nil),
				"balanceOf", user1.AddressBase58Check)

			By("BACKWARD")

			By("channel transfer by customer backward")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelCC, cmn.ChannelCC, admin, "channelTransferByAdmin", "",
				client.NewNonceByTime().Get(), nil, id2, "FIAT", user1.AddressBase58Check, "FIAT", transferAmount)

			By("check allowed balance after transfer")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("get channel transfer from")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrFrom, nil),
				"channelTransferFrom", id2)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			client.TxInvokeByRobot(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, nil, "createCCTransferTo", from)

			By("check fiat balance 1")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
				"balanceOf", user1.AddressBase58Check)

			By("channel transfer to")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrTo, nil),
				"channelTransferTo", id2)

			By("commit cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2)

			By("delete cc transfer to")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferTo", id2)

			By("delete cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2)

			By("check allowed balance")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("check fiat balance 2")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
				"balanceOf", user1.AddressBase58Check)
		})

		It("cancel forward success", func() {
			By("cancel channel transfer forward")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id, "CC", "FIAT", transferAmount)

			By("check balance after transfer")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(balanceAfterTransfer), nil),
				"balanceOf", user1.AddressBase58Check)

			By("get channel transfer from")
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
				"channelTransferFrom", id)

			By("cancel cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat, "cancelCCTransferFrom", id)

			By("check allowed balance 2")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("check fiat balance")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
				"balanceOf", user1.AddressBase58Check)
		})

		It("cancel backward success", func() {
			By("FORWARD")

			By("channel transfer by customer forward")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id, "CC", "FIAT", transferAmount)

			By("check balance after transfer")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(balanceAfterTransfer), nil),
				"balanceOf", user1.AddressBase58Check)

			By("get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)
				return ""
			}
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
				"channelTransferFrom", id)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			client.TxInvokeByRobot(network, peer, network.Orderers[0],
				cmn.ChannelCC, cmn.ChannelCC, nil, "createCCTransferTo", from)

			By("check allowed balance 1")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(transferAmount), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
				"channelTransferTo", id)

			By("commit cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat, "commitCCTransferFrom", id)

			By("delete cc transfer to")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferTo", id)

			By("delete cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferFrom", id)

			By("check allowed balance 2")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(transferAmount), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("check fiat balance")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(balanceAfterTransfer), nil),
				"balanceOf", user1.AddressBase58Check)

			By("BACKWARD")

			By("channel transfer by customer backward")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelCC, cmn.ChannelCC, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id2, "FIAT", "FIAT", transferAmount)

			By("check allowed balance after transfer")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("get channel transfer from")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fChTrTo, nil), "channelTransferFrom", id2)

			By("cancel cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "cancelCCTransferFrom", id2)

			By("check allowed balance 2")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(transferAmount), nil),
				"allowedBalanceOf", user1.AddressBase58Check, "FIAT")

			By("check fiat balance")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(balanceAfterTransfer), nil),
				"balanceOf", user1.AddressBase58Check)
		})

		It("query all transfers from", func() {
			transferAmount = "100"
			ids := make(map[string]struct{})

			By("channel transfer by customer forward1")
			id = uuid.NewString()
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id, "CC", "FIAT", transferAmount)
			ids[id] = struct{}{}

			By("channel transfer by customer forward2")
			id = uuid.NewString()
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id, "CC", "FIAT", transferAmount)
			ids[id] = struct{}{}

			By("channel transfer by customer forward3")
			id = uuid.NewString()
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id, "CC", "FIAT", transferAmount)
			ids[id] = struct{}{}

			By("channel transfer by customer forward4")
			id = uuid.NewString()
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id, "CC", "FIAT", transferAmount)
			ids[id] = struct{}{}

			By("channel transfer by customer forward5")
			id = uuid.NewString()
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id, "CC", "FIAT", transferAmount)
			ids[id] = struct{}{}

			By("check balance after transfer")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("500"), nil),
				"balanceOf", user1.AddressBase58Check)

			bookmark := ""
			By("checking size")
			fSize := func(out []byte) string {
				resp := pbfound.CCTransfers{}
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Bookmark).ToNot(BeEmpty())
				Expect(resp.Ccts).To(HaveLen(2))

				return ""
			}
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fSize, nil),
				"channelTransfersFrom", "2", bookmark)

			By("checking size")
			bookmark = ""
			fCheckIds := func(out []byte) string {
				resp := pbfound.CCTransfers{}
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Bookmark).To(BeEmpty())
				Expect(resp.Ccts).To(HaveLen(5))
				for _, cct := range resp.Ccts {
					Expect(ids).Should(HaveKey(cct.Id))
				}

				return ""
			}
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fCheckIds, nil),
				"channelTransfersFrom", "1000", bookmark)

			count := 0
			bookmark = ""
			for {
				fCheckBookmark := func(out []byte) string {
					resp := pbfound.CCTransfers{}
					err := json.Unmarshal(out, &resp)
					Expect(err).NotTo(HaveOccurred())
					bookmark = resp.Bookmark
					return ""
				}

				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fCheckBookmark, nil),
					"channelTransfersFrom", "2", bookmark)

				if bookmark == "" {
					Expect(count).To(Equal(2))
					break
				}

				count++
			}
		})

		It("multi transfer by customer success", func() {
			By("FORWARD")

			By("FORWARD. channel transfer by customer forward")
			forwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, "channelMultiTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id, "CC", string(forwardItemsJSON))

			By("FORWARD. check industrial balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, "0"), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}

			By("FORWARD. check cc allowed balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("FORWARD. get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)
				return ""
			}
			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial, fabricnetwork.CheckResult(fChTrFrom, nil),
				"channelTransferFrom", id)
			Expect(from).NotTo(BeEmpty())

			By("FORWARD. create cc transfer to")
			client.TxInvokeByRobot(network, peer, network.Orderers[0],
				cmn.ChannelCC, cmn.ChannelCC, nil, "createCCTransferTo", from)

			By("FORWARD. check industrial balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, "0"), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}

			By("FORWARD. check cc allowed balance after createCCTransferTo")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(item.Amount.String()), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("FORWARD. channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
				"channelTransferTo", id)

			By("FORWARD. commit cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, "commitCCTransferFrom", id)

			By("FORWARD. delete cc transfer to")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferTo", id)

			By("FORWARD. delete cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, "deleteCCTransferFrom", id)

			By("FORWARD. check cc allowed balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(item.Amount.String()), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("FORWARD. check industrial balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, "0"), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}

			By("BACKWARD")

			By("BACKWARD. channel transfer by customer backward")
			backwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelCC, cmn.ChannelCC, user1, "channelMultiTransferByCustomer", "",
				client.NewNonceByTime().Get(), nil, id2, "INDUSTRIAL", string(backwardItemsJSON))

			By("BACKWARD. check cc allowed balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("BACKWARD. check industrial balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, "0"), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}

			By("BACKWARD. get channel transfer from")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrFrom, nil),
				"channelTransferFrom", id2)
			Expect(from).NotTo(BeEmpty())

			By("BACKWARD. create cc transfer to")
			client.TxInvokeByRobot(network, peer, network.Orderers[0],
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, nil, "createCCTransferTo", from)

			By("BACKWARD. check industrial allowed balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, item.Amount.String()), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}
			By("BACKWARD. check cc allowed balance after createCCTransferTo")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("BACKWARD. channel transfer to")
			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial, fabricnetwork.CheckResult(fChTrTo, nil),
				"channelTransferTo", id2)

			By("BACKWARD. commit cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2)

			By("BACKWARD. delete cc transfer to")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, "deleteCCTransferTo", id2)

			By("BACKWARD. delete cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2)

			By("BACKWARD. check cc allowed balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("BACKWARD. check industrial balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, item.Amount.String()), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}
		})

		It("multi transfer by admin success", func() {
			By("FORWARD")

			By("FORWARD. channel transfer forward")
			forwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, admin, "channelMultiTransferByAdmin", "",
				client.NewNonceByTime().Get(), nil, id, "CC", user1.AddressBase58Check, string(forwardItemsJSON))

			By("FORWARD. check industrial balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, "0"), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}

			By("FORWARD. check cc allowed balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("FORWARD. get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)

				return ""
			}
			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial, fabricnetwork.CheckResult(fChTrFrom, nil),
				"channelTransferFrom", id)
			Expect(from).NotTo(BeEmpty())

			By("FORWARD. create cc transfer to")
			client.TxInvokeByRobot(network, peer, network.Orderers[0],
				cmn.ChannelCC, cmn.ChannelCC, nil, "createCCTransferTo", from)

			By("FORWARD. check industrial balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, "0"), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}

			By("FORWARD. check cc allowed after createCCTransferTo")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(item.Amount.String()), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("FORWARD. channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
				"channelTransferTo", id)

			By("FORWARD. commit cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, "commitCCTransferFrom", id)

			By("FORWARD. delete cc transfer to")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferTo", id)

			By("FORWARD. delete cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, "deleteCCTransferFrom", id)

			By("FORWARD. check cc allowed balance. after deleteCCTransferFrom")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(item.Amount.String()), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("FORWARD. check industrial allowed balance. after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, "0"), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}

			By("BACKWARD")

			By("BACKWARD. channel transfer by customer backward")
			backwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelCC, cmn.ChannelCC, admin, "channelMultiTransferByAdmin", "",
				client.NewNonceByTime().Get(), nil, id2, "INDUSTRIAL", user1.AddressBase58Check, string(backwardItemsJSON))

			By("BACKWARD. check cc allowed balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("BACKWARD. check industrial balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, "0"), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}

			By("BACKWARD. get channel transfer from")
			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrFrom, nil),
				"channelTransferFrom", id2)
			Expect(from).NotTo(BeEmpty())

			By("BACKWARD. create cc transfer to")
			client.TxInvokeByRobot(network, peer, network.Orderers[0],
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, nil, "createCCTransferTo", from)

			By("BACKWARD. check cc balance after createCCTransferTo")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("BACKWARD. check industrial balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, item.Amount.String()), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}

			By("BACKWARD. channel transfer to")
			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial, fabricnetwork.CheckResult(fChTrTo, nil),
				"channelTransferTo", id2)

			By("BACKWARD. commit cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2)

			By("BACKWARD. delete cc transfer to")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, "deleteCCTransferTo", id2)

			By("BACKWARD. delete cc transfer from")
			client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
				cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2)

			By("BACKWARD. check cc allowed balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
					"allowedBalanceOf", user1.AddressBase58Check, item.Token)
			}

			By("BACKWARD. check industrial balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, item.Amount.String()), nil),
					"industrialBalanceOf", user1.AddressBase58Check)
			}
		})
	})
})
