package integration

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	pb "github.com/anoideaopen/foundation/proto"
	industrialtoken "github.com/anoideaopen/foundation/test/chaincode/industrial/industrial_token"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	"github.com/btcsuite/btcutil/base58"
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
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
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
		testDir, err = os.MkdirTemp("", "foundation")
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

	Describe("smartbft standart test", func() {
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

	Describe("foundation test", func() {
		var (
			channels       = []string{"acl", "cc", "fiat", "industrial"}
			ordererRunners []*ginkgomon.Runner
			redisProcess   ifrit.Process
			redisDB        *runner.RedisDB
			networkFound   *cmn.NetworkFoundation
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
			redisProcess.Signal(syscall.SIGTERM)
			Eventually(redisProcess.Wait(), time.Minute).Should(Receive())
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

			network = nwo.New(networkConfig, testDir, client, StartPort(), components)
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
		})

		It("deploy acl cc fiat industrial and query", func() {
			By("Deploying chaincode")
			peer := network.Peer("Org1", "peer0")

			pathToPrivateKeyBackend := network.PeerUserKey(peer, "User1")
			skiBackend, err := cmn.ReadSKI(pathToPrivateKeyBackend)
			Expect(err).NotTo(HaveOccurred())

			pathToPrivateKeyRobot := network.PeerUserKey(peer, "User2")
			skiRobot, err := cmn.ReadSKI(pathToPrivateKeyRobot)
			Expect(err).NotTo(HaveOccurred())

			validators, err := cmn.NewSecrets(1)
			Expect(err).NotTo(HaveOccurred())
			adminPriv := validators[0]
			adminPub := adminPriv.Public().(ed25519.PublicKey)
			adminHash := sha3.Sum256(adminPub)
			adminAddr := base58.CheckEncode(adminHash[1:], adminHash[0])

			By("Deploying chaincode acl")
			ctorAcl := fmt.Sprintf(
				`{"Args":["%s","%d","%s"]}`, skiBackend, 1,
				base58.Encode(validators[0].Public().(ed25519.PublicKey)),
			)
			cmn.DeployChaincodeACL(network, components, ctorAcl, testDir)

			By("querying the chaincode from acl")
			sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
				ChannelID: "acl",
				Name:      "acl",
				Ctor:      `{"Args":["getAddresses", "10", ""]}`,
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
			Eventually(sess, network.EventuallyTimeout).Should(gbytes.Say(`{"Addrs":null,"Bookmark":""}`))

			By("Deploying chaincode cc")
			cfgCC := &pb.Config{
				Contract: &pb.ContractConfig{Symbol: "CC", RobotSKI: skiRobot,
					Admin: &pb.Wallet{Address: adminAddr}},
				Token: &pb.TokenConfig{Name: "Currency Coin", Decimals: 8,
					UnderlyingAsset: "US Dollars", Issuer: &pb.Wallet{Address: adminAddr}},
			}
			cfgBytesCC, err := protojson.Marshal(cfgCC)
			Expect(err).NotTo(HaveOccurred())
			ctorCC := fmt.Sprintf(`{"Args":[%s]}`, strconv.Quote(string(cfgBytesCC)))
			cmn.DeployChaincodeFoundation(network, "cc", components,
				"github.com/anoideaopen/foundation/test/chaincode/cc", ctorCC, testDir)

			By("querying the chaincode from cc")
			sess, err = network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
				ChannelID: "cc",
				Name:      "cc",
				Ctor:      `{"Args":["metadata"]}`,
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
			Eventually(sess, network.EventuallyTimeout).Should(gbytes.Say(`{"name":"Currency Coin","symbol":"CC","decimals":8,"underlying_asset":"US Dollars"`))

			By("Deploying chaincode fiat")
			feeUsers, err := cmn.NewSecrets(2)
			feeSetPriv := feeUsers[0]
			feeSetPub := feeSetPriv.Public().(ed25519.PublicKey)
			feeSetHash := sha3.Sum256(feeSetPub)
			feeSetAddr := base58.CheckEncode(feeSetHash[1:], feeSetHash[0])
			feeAdrPriv := feeUsers[1]
			feeAdrPub := feeAdrPriv.Public().(ed25519.PublicKey)
			feeAdrHash := sha3.Sum256(feeAdrPub)
			feeAdrAddr := base58.CheckEncode(feeAdrHash[1:], feeAdrHash[0])
			cfgFiat := &pb.Config{
				Contract: &pb.ContractConfig{
					Symbol:   "FIAT",
					RobotSKI: skiRobot,
					Admin:    &pb.Wallet{Address: adminAddr},
					Options: &pb.ChaincodeOptions{
						DisabledFunctions: []string{"TxBuyToken", "TxBuyBack"},
					},
				},
				Token: &pb.TokenConfig{
					Name:             "FIAT",
					Decimals:         8,
					UnderlyingAsset:  "US Dollars",
					Issuer:           &pb.Wallet{Address: adminAddr},
					FeeSetter:        &pb.Wallet{Address: feeSetAddr},
					FeeAddressSetter: &pb.Wallet{Address: feeAdrAddr},
				},
			}
			cfgBytesFiat, err := protojson.Marshal(cfgFiat)
			Expect(err).NotTo(HaveOccurred())
			ctorFiat := fmt.Sprintf(`{"Args":[%s]}`, strconv.Quote(string(cfgBytesFiat)))
			cmn.DeployChaincodeFoundation(network, "fiat", components,
				"github.com/anoideaopen/foundation/test/chaincode/fiat", ctorFiat, testDir)

			By("querying the chaincode from fiat")
			sess, err = network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
				ChannelID: "fiat",
				Name:      "fiat",
				Ctor:      `{"Args":["metadata"]}`,
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
			Eventually(sess, network.EventuallyTimeout).Should(gbytes.Say(`{"name":"FIAT","symbol":"FIAT","decimals":8,"underlying_asset":"US Dollars"`))

			extCfg := industrialtoken.ExtConfig{
				Name:             "Industrial token",
				Decimals:         8,
				UnderlyingAsset:  "TEST_UnderlyingAsset",
				DeliveryForm:     "TEST_DeliveryForm",
				UnitOfMeasure:    "TEST_IT",
				TokensForUnit:    "1",
				PaymentTerms:     "Non-prepaid",
				Price:            "Floating",
				Issuer:           &pb.Wallet{Address: adminAddr},
				FeeSetter:        &pb.Wallet{Address: feeSetAddr},
				FeeAddressSetter: &pb.Wallet{Address: feeAdrAddr},
			}
			cfgIndustrial := &pb.Config{
				Contract: &pb.ContractConfig{
					Symbol:   "INDUSTRIAL",
					RobotSKI: skiRobot,
					Admin:    &pb.Wallet{Address: adminAddr},
				},
			}
			cfgIndustrial.ExtConfig, _ = anypb.New(&extCfg)

			cfgBytesIndustrial, err := protojson.Marshal(cfgIndustrial)
			Expect(err).NotTo(HaveOccurred())
			ctorIndustrial := fmt.Sprintf(`{"Args":[%s]}`, strconv.Quote(string(cfgBytesIndustrial)))
			By("Deploying chaincode industrial")
			cmn.DeployChaincodeFoundation(network, "industrial", components,
				"github.com/anoideaopen/foundation/test/chaincode/industrial", ctorIndustrial, testDir)

			By("querying the chaincode from industrial")
			sess, err = network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
				ChannelID: "industrial",
				Name:      "industrial",
				Ctor:      `{"Args":["metadata"]}`,
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
			Eventually(sess, network.EventuallyTimeout).Should(gbytes.Say(`{"name":"Industrial token","symbol":"INDUSTRIAL","decimals":8,"underlying_asset":"TEST_UnderlyingAsset"`))

			By("start robot")
			robotRunner := networkFound.RobotRunner()
			robotRunner.Command.Env = append(robotRunner.Command.Env, "FABRIC_LOGGING_SPEC=orderer.consensus.smartbft=debug:grpc=debug")
			robotProc := ifrit.Invoke(robotRunner)
			Eventually(robotProc.Ready(), network.EventuallyTimeout).Should(BeClosed())
		})
	})
})

func peerGroupRunners(n *nwo.Network) (ifrit.Runner, []*ginkgomon.Runner) {
	var runners []*ginkgomon.Runner
	members := grouper.Members{}
	for _, p := range n.Peers {
		peerRunner := n.PeerRunner(p)
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
