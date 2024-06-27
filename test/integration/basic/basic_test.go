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
	"github.com/bsm/gomega/gexec"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
)

const fnMethodWithRights = "withRights"

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

		It("add user", func() {
			user, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
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
			user, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			client.AddUser(network, peer, network.Orderers[0], user)

			By("send a request that is similar to invoke")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("Ok"), nil),
				"allowedBalanceAdd", "CC", user.AddressBase58Check, "50", "add some assets")

			By("let's check the allowed balance - 1")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
				"allowedBalanceOf", user.AddressBase58Check, "CC")

			By("send an invoke that is similar to request")
			client.NBTxInvoke(network, peer, network.Orderers[0], nil,
				cmn.ChannelFiat, cmn.ChannelFiat,
				"allowedBalanceAdd", "CC", user.AddressBase58Check, "50", "add some assets")

			By("let's check the allowed balance - 2")
			client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
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
				var err error

				user1, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())
				user2, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())
			})

			It("transfer", func() {
				By("add users to acl")
				client.AddUser(network, peer, network.Orderers[0], user1)
				client.AddUser(network, peer, network.Orderers[0], user2)

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

			It("transfer with fee", func() {
				By("add users to acl")
				user1.UserID = "1111"
				user2.UserID = "2222"

				client.AddUser(network, peer, network.Orderers[0], user1)
				client.AddUser(network, peer, network.Orderers[0], user2)
				client.AddUser(network, peer, network.Orderers[0], feeSetter)
				client.AddUser(network, peer, network.Orderers[0], feeAddressSetter)

				feeWallet, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())

				client.AddUser(network, peer, network.Orderers[0], feeWallet)

				By("emit tokens")
				amount := "3"
				amountOne := "1"
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, admin,
					"emit", "", client.NewNonceByTime().Get(), nil, user1.AddressBase58Check, amount)

				By("emit check")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("set fee")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeSetter,
					"setFee", "", client.NewNonceByTime().Get(), nil, "FIAT", "1", "1", "100")

				By("set fee address")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeAddressSetter,
					"setFeeAddress", "", client.NewNonceByTime().Get(), nil, feeWallet.AddressBase58Check)

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
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fFeeTransfer, nil),
					"getFeeTransfer", string(bytes))

				By("transfer tokens from user1 to user2")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), nil, user2.AddressBase58Check, amountOne, "ref transfer")

				By("check balance user1")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amountOne), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance user2")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amountOne), nil),
					"balanceOf", user2.AddressBase58Check)

				By("check balance feeWallet")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amountOne), nil),
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

				feeWallet, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())

				client.AddUser(network, peer, network.Orderers[0], feeWallet)

				By("emit tokens")
				amount := "3"
				amountOne := "1"
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, admin,
					"emit", "", client.NewNonceByTime().Get(), nil, user1.AddressBase58Check, amount)

				By("emit check")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("set fee")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeSetter,
					"setFee", "", client.NewNonceByTime().Get(), nil, "FIAT", "1", "1", "100")

				By("set fee address")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeAddressSetter,
					"setFeeAddress", "", client.NewNonceByTime().Get(), nil, feeWallet.AddressBase58Check)

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
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fFeeTransfer, nil),
					"getFeeTransfer", string(bytes))

				By("transfer tokens from user1 to user2")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), nil, user2.AddressBase58Check, amountOne, "ref transfer")

				By("check balance user1")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("2"), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance user2")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amountOne), nil),
					"balanceOf", user2.AddressBase58Check)

				By("check balance feeWallet")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
					"balanceOf", feeWallet.AddressBase58Check)
			})

			It("transfer to the same wallet with fee is on", func() {
				By("add users to acl")
				client.AddUser(network, peer, network.Orderers[0], user1)
				client.AddUser(network, peer, network.Orderers[0], feeSetter)
				client.AddUser(network, peer, network.Orderers[0], feeAddressSetter)

				feeWallet, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())

				client.AddUser(network, peer, network.Orderers[0], feeWallet)

				By("emit tokens")
				amount := "3"
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, admin,
					"emit", "", client.NewNonceByTime().Get(), nil, user1.AddressBase58Check, amount)

				By("emit check")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("set fee")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeSetter,
					"setFee", "", client.NewNonceByTime().Get(), nil, "FIAT", "1", "1", "100")

				By("set fee address")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, feeAddressSetter,
					"setFeeAddress", "", client.NewNonceByTime().Get(), nil, feeWallet.AddressBase58Check)

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
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fFeeTransfer, nil),
					"getFeeTransfer", string(bytes))

				By("NEGATIVE: transfer tokens from user1 to user2")
				client.TxInvokeWithSign(network, peer, network.Orderers[0],
					cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), fabricnetwork.CheckResult(nil, fabricnetwork.CheckTxResponseResult("TxTransfer: sender and recipient are same users")), user1.AddressBase58Check, "1", "ref transfer")

				By("check balance user1")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance feeWallet")
				client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
					fabricnetwork.CheckResult(fabricnetwork.CheckBalance("0"), nil),
					"balanceOf", feeWallet.AddressBase58Check)
			})
		})

		It("accessmatrix - add and remove rights", func() {
			By("add user to acl")
			user1, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			client.AddUser(network, peer, network.Orderers[0], user1)

			user2, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			client.AddUser(network, peer, network.Orderers[0], user2)

			By("invoking industrial chaincode with user have no rights")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, fnMethodWithRights, "",
				client.NewNonceByTime().Get(), fabricnetwork.CheckResult(nil, fabricnetwork.CheckTxResponseResult("unauthorized")))

			By("add rights and check rights")
			client.AddRights(network, peer, network.Orderers[0],
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, "issuer", "", user1)

			By("invoking industrial chaincode with acl right user")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, fnMethodWithRights, "",
				client.NewNonceByTime().Get(), nil)

			By("remove rights and check rights")
			client.RemoveRights(network, peer, network.Orderers[0],
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, "issuer", "", user1)

			By("invoking industrial chaincode with user acl rights removed")
			client.TxInvokeWithSign(network, peer, network.Orderers[0],
				cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, fnMethodWithRights, "",
				client.NewNonceByTime().Get(), fabricnetwork.CheckResult(nil, fabricnetwork.CheckTxResponseResult("unauthorized")))

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
