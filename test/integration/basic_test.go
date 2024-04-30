package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/integration/channelparticipation"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
	"github.com/tedsuo/ifrit/grouper"
)

var _ = Describe("Basic foundation Tests", func() {
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

	Describe("smartbft standart test", func() {
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
				Ctor:      cmn.CtorFromSlice([]string{"query", "a"}),
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

			peerGroupRunner, _ := peerGroupRunners(network)
			peerProcesses = ifrit.Invoke(peerGroupRunner)
			Eventually(peerProcesses.Ready(), network.EventuallyTimeout).Should(BeClosed())

			By("Joining orderers to channels")
			for _, channel := range channels {
				joinChannel(network, channel)
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

		It("add user", func() {
			user := client.NewUserFoundation()
			client.AddUser(network, peer, network.Orderers[0], user)
		})

		It("check metadata in chaincode", func() {
			By("querying the chaincode from cc")
			sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
				ChannelID: cmn.ChannelCC,
				Name:      cmn.ChannelCC,
				Ctor:      cmn.CtorFromSlice([]string{"metadata"}),
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
			Eventually(sess, network.EventuallyTimeout).Should(gbytes.Say(`{"name":"Currency Coin","symbol":"CC","decimals":8,"underlying_asset":"US Dollars"`))

			By("querying the chaincode from fiat")
			sess, err = network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
				ChannelID: cmn.ChannelFiat,
				Name:      cmn.ChannelFiat,
				Ctor:      cmn.CtorFromSlice([]string{"metadata"}),
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
			Eventually(sess, network.EventuallyTimeout).Should(gbytes.Say(`{"name":"FIAT","symbol":"FIAT","decimals":8,"underlying_asset":"US Dollars"`))

			By("querying the chaincode from industrial")
			sess, err = network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
				ChannelID: cmn.ChannelIndustrial,
				Name:      cmn.ChannelIndustrial,
				Ctor:      cmn.CtorFromSlice([]string{"metadata"}),
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
			Eventually(sess, network.EventuallyTimeout).Should(gbytes.Say(`{"name":"Industrial token","symbol":"INDUSTRIAL","decimals":8,"underlying_asset":"TEST_UnderlyingAsset"`))
		})

		It("query test", func() {
			user := client.NewUserFoundation()
			client.AddUser(network, peer, network.Orderers[0], user)

			By("send a request that is similar to invoke")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				checkResult(checkBalance("Ok"), nil),
				"allowedBalanceAdd", "CC", user.AddressBase58Check, "50", "add some assets")

			By("let's check the allowed balance - 1")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				checkResult(checkBalance("0"), nil),
				"allowedBalanceOf", user.AddressBase58Check, "CC")

			By("send a invoke that is similar to request")
			client.NBTxInvoke(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat,
				"allowedBalanceAdd", "CC", user.AddressBase58Check, "50", "add some assets")

			By("let's check the allowed balance - 2")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				checkResult(checkBalance("0"), nil),
				"allowedBalanceOf", user.AddressBase58Check, "CC")
		})

		Describe("transfer tests", func() {
			var (
				user1 *client.UserFoundation
				user2 *client.UserFoundation
			)

			BeforeEach(func() {
				By("add admin to acl")
				client.AddUser(network, peer, network.Orderers[0], admin)

				By("create users")
				user1 = client.NewUserFoundation()
				user2 = client.NewUserFoundation()
			})

			It("transfer", func() {
				By("add users to acl")
				client.AddUser(network, peer, network.Orderers[0], user1)
				client.AddUser(network, peer, network.Orderers[0], user2)

				By("emit tokens")
				amount := "1"
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, admin,
					"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, amount)

				By("emit check")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance(amount), nil),
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
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, checkResult(nil, fErr),
					"getFeeTransfer", string(bytes))

				By("transfer tokens from user1 to user2")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), user2.AddressBase58Check, amount, "ref transfer")

				By("check balance user1")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance("0"), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance user2")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance(amount), nil),
					"balanceOf", user2.AddressBase58Check)
			})

			It("transfer with fee", func() {
				By("add users to acl")
				user1.UserID = "1111"
				user2.UserID = "2222"

				client.AddUser(network, peer, network.Orderers[0], user1)
				client.AddUser(network, peer, network.Orderers[0], user2)
				client.AddUser(network, peer, network.Orderers[0], feeSetter)
				client.AddUser(network, peer, network.Orderers[0], feeAddressSetter)

				feeWallet := client.NewUserFoundation()
				client.AddUser(network, peer, network.Orderers[0], feeWallet)

				By("emit tokens")
				amount := "3"
				amountOne := "1"
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, admin,
					"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, amount)

				By("emit check")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("set fee")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeSetter,
					"setFee", "", client.NewNonceByTime().Get(), "FIAT", "1", "1", "100")

				By("set fee address")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeAddressSetter,
					"setFeeAddress", "", client.NewNonceByTime().Get(), feeWallet.AddressBase58Check)

				By("get transfer fee from user1 to user2")
				req := FeeTransferRequestDTO{
					SenderAddress:    user1.AddressBase58Check,
					RecipientAddress: user2.AddressBase58Check,
					Amount:           amount,
				}
				bytes, err := json.Marshal(req)
				Expect(err).NotTo(HaveOccurred())

				fFeeTransfer := func(out []byte) string {
					resp := FeeTransferResponseDTO{}
					err = json.Unmarshal(out, &resp)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.FeeAddress).To(Equal(feeWallet.AddressBase58Check))
					Expect(resp.Amount).To(Equal("1"))
					Expect(resp.Currency).To(Equal("FIAT"))

					return ""
				}
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, checkResult(fFeeTransfer, nil),
					"getFeeTransfer", string(bytes))

				By("transfer tokens from user1 to user2")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), user2.AddressBase58Check, amountOne, "ref transfer")

				By("check balance user1")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance(amountOne), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance user2")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance(amountOne), nil),
					"balanceOf", user2.AddressBase58Check)

				By("check balance feeWallet")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance(amountOne), nil),
					"balanceOf", feeWallet.AddressBase58Check)
			})

			It("transfer to itself to second wallet with fee is on", func() {
				By("add users to acl")
				user1.UserID = "1111"
				user2.UserID = "1111"

				client.AddUser(network, peer, network.Orderers[0], user1)
				client.AddUser(network, peer, network.Orderers[0], user2)
				client.AddUser(network, peer, network.Orderers[0], feeSetter)
				client.AddUser(network, peer, network.Orderers[0], feeAddressSetter)

				feeWallet := client.NewUserFoundation()
				client.AddUser(network, peer, network.Orderers[0], feeWallet)

				By("emit tokens")
				amount := "3"
				amountOne := "1"
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, admin,
					"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, amount)

				By("emit check")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("set fee")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeSetter,
					"setFee", "", client.NewNonceByTime().Get(), "FIAT", "1", "1", "100")

				By("set fee address")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeAddressSetter,
					"setFeeAddress", "", client.NewNonceByTime().Get(), feeWallet.AddressBase58Check)

				By("get transfer fee from user1 to user2")
				req := FeeTransferRequestDTO{
					SenderAddress:    user1.AddressBase58Check,
					RecipientAddress: user2.AddressBase58Check,
					Amount:           amountOne,
				}
				bytes, err := json.Marshal(req)
				Expect(err).NotTo(HaveOccurred())

				fFeeTransfer := func(out []byte) string {
					resp := FeeTransferResponseDTO{}
					err = json.Unmarshal(out, &resp)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.FeeAddress).To(Equal(feeWallet.AddressBase58Check))
					Expect(resp.Amount).To(Equal("0"))
					Expect(resp.Currency).To(Equal("FIAT"))

					return ""
				}
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, checkResult(fFeeTransfer, nil),
					"getFeeTransfer", string(bytes))

				By("transfer tokens from user1 to user2")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), user2.AddressBase58Check, amountOne, "ref transfer")

				By("check balance user1")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance("2"), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance user2")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance(amountOne), nil),
					"balanceOf", user2.AddressBase58Check)

				By("check balance feeWallet")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance("0"), nil),
					"balanceOf", feeWallet.AddressBase58Check)
			})

			It("transfer to the same wallet with fee is on", func() {
				By("add users to acl")
				client.AddUser(network, peer, network.Orderers[0], user1)
				client.AddUser(network, peer, network.Orderers[0], feeSetter)
				client.AddUser(network, peer, network.Orderers[0], feeAddressSetter)

				feeWallet := client.NewUserFoundation()
				client.AddUser(network, peer, network.Orderers[0], feeWallet)

				By("emit tokens")
				amount := "3"
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, admin,
					"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, amount)

				By("emit check")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("set fee")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeSetter,
					"setFee", "", client.NewNonceByTime().Get(), "FIAT", "1", "1", "100")

				By("set fee address")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeAddressSetter,
					"setFeeAddress", "", client.NewNonceByTime().Get(), feeWallet.AddressBase58Check)

				By("get transfer fee from user1 to user2")
				req := FeeTransferRequestDTO{
					SenderAddress:    user1.AddressBase58Check,
					RecipientAddress: user1.AddressBase58Check,
					Amount:           "450",
				}
				bytes, err := json.Marshal(req)
				Expect(err).NotTo(HaveOccurred())

				fFeeTransfer := func(out []byte) string {
					resp := FeeTransferResponseDTO{}
					err = json.Unmarshal(out, &resp)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.FeeAddress).To(Equal(feeWallet.AddressBase58Check))
					Expect(resp.Amount).To(Equal("0"))
					Expect(resp.Currency).To(Equal("FIAT"))

					return ""
				}
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, checkResult(fFeeTransfer, nil),
					"getFeeTransfer", string(bytes))

				By("transfer tokens from user1 to user2")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), user1.AddressBase58Check, "1", "ref transfer")

				By("check balance user1")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance feeWallet")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					checkResult(checkBalance("0"), nil),
					"balanceOf", feeWallet.AddressBase58Check)
			})
		})

		It("accessmatrix - add and remove rights", func() {
			By("add user to acl")
			user1 := client.NewUserFoundation()
			client.AddUser(network, peer, network.Orderers[0], user1)

			By("add rights and check rights")
			client.AddRights(network, peer, network.Orderers[0],
				cmn.ChannelAcl, cmn.ChannelAcl, "issuer", "testOperation", user1)

			By("remove rights and check rights")
			client.RemoveRights(network, peer, network.Orderers[0],
				cmn.ChannelAcl, cmn.ChannelAcl, "issuer", "testOperation", user1)
		})

		It("example test", func() {
			By("add admin to acl")
			client.AddUser(network, peer, network.Orderers[0], admin)

			By("add user to acl")
			user1 := client.NewUserFoundation()
			client.AddUser(network, peer, network.Orderers[0], user1)

			By("emit tokens")
			emitAmount := "1"
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelFiat, cmn.ChannelFiat, admin,
				"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, emitAmount)

			By("emit check")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				checkResult(checkBalance(emitAmount), nil),
				"balanceOf", user1.AddressBase58Check)
		})
	})
})

type FeeTransferRequestDTO struct {
	SenderAddress    string `json:"sender_address,omitempty"`
	RecipientAddress string `json:"recipient_address,omitempty"`
	Amount           string `json:"amount,omitempty"`
}

type FeeTransferResponseDTO struct {
	FeeAddress string `json:"fee_address,omitempty"`
	Amount     string `json:"amount,omitempty"`
	Currency   string `json:"currency,omitempty"`
}

func checkResult(successF func(out []byte) string, errorF func(outErr []byte) string) client.CheckResultFunc {
	return func(err error, sessExitCode int, sessError []byte, sessOut []byte) string {
		if (successF == nil && errorF == nil) ||
			(successF != nil && errorF != nil) {
			return "error: only one function must be defined"
		}

		if err != nil {
			return fmt.Sprintf("error executing command: %v", err)
		}

		if successF != nil {
			if sessExitCode != 0 {
				return fmt.Sprintf("exit code is %d: %s, %v", sessExitCode, string(sessError), err)
			}
			out := sessOut[:len(sessOut)-1] // skip line feed
			return successF(out)
		}

		if sessExitCode == 0 {
			return fmt.Sprintf("exit code is %d", sessExitCode)
		}

		return errorF(sessError)
	}
}

func checkBalance(etalon string) func([]byte) string {
	return func(out []byte) string {
		etl := "\"" + etalon + "\""
		if string(out) != etl {
			return "not equal " + string(out) + " and " + etl
		}
		return ""
	}
}

func peerGroupRunners(n *nwo.Network) (ifrit.Runner, []*ginkgomon.Runner) {
	var runners []*ginkgomon.Runner
	members := grouper.Members{}
	for _, p := range n.Peers {
		peerRunner := n.PeerRunner(p, "FABRIC_LOGGING_SPEC=debug:grpc=debug")
		members = append(members, grouper.Member{Name: p.ID(), Runner: peerRunner})
		runners = append(runners, peerRunner)
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
		Ctor:            cmn.CtorFromSlice([]string{"init", "a", "100", "b", "200"}),
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
		Ctor:      cmn.CtorFromSlice([]string{"invoke", "a", "b", "10"}),
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
			Ctor:      cmn.CtorFromSlice([]string{"query", key}),
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
