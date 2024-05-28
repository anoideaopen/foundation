package integration

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	cligrpc "github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	"github.com/btcsuite/btcutil/base58"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ = Describe("Channel transfer foundation Tests", func() {
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

	var (
		channels            = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
		ordererRunners      []*ginkgomon.Runner
		redisProcess        ifrit.Process
		redisDB             *runner.RedisDB
		networkFound        *cmn.NetworkFoundation
		robotProc           ifrit.Process
		channelTransferProc ifrit.Process
		skiBackend          string
		skiRobot            string
		peer                *nwo.Peer
		admin               *client.UserFoundation
		feeSetter           *client.UserFoundation
		feeAddressSetter    *client.UserFoundation
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
		networkFound.ChannelTransfer.RedisAddresses = []string{redisDB.Address()}

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

		By("start channel transfer")
		channelTransferRunner := networkFound.ChannelTransferRunner()
		channelTransferProc = ifrit.Invoke(channelTransferRunner)
		Eventually(channelTransferProc.Ready(), network.EventuallyTimeout).Should(BeClosed())
	})
	AfterEach(func() {
		By("stop robot")
		if robotProc != nil {
			robotProc.Signal(syscall.SIGTERM)
			Eventually(robotProc.Wait(), network.EventuallyTimeout).Should(Receive())
		}

		By("stop channel transfer")
		if channelTransferProc != nil {
			channelTransferProc.Signal(syscall.SIGTERM)
			Eventually(channelTransferProc.Wait(), network.EventuallyTimeout).Should(Receive())
		}
	})
	It("example test", func() {
		By("add admin to acl")
		client.AddUser(network, peer, network.Orderers[0], admin)

		By("add user to acl")
		user1 := client.NewUserFoundation()
		client.AddUser(network, peer, network.Orderers[0], user1)

		By("emit tokens")
		emitAmount := "1000"
		client.TxInvokeWithSign(network, peer, network.Orderers[0],
			cmn.ChannelFiat, cmn.ChannelFiat, admin,
			"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, emitAmount)

		By("emit check")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			checkResult(checkBalance(emitAmount), nil),
			"balanceOf", user1.AddressBase58Check)

		By("creating grpc connection")
		clientCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)
		conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(transportCredentials))
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}()

		By("creating channel transfer API client")
		c := cligrpc.NewAPIClient(conn)

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, "CC", user1.AddressBase58Check, "FIAT", "250"}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{"channelTransferByAdmin", requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := admin.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transfer := &cligrpc.TransferBeginAdminRequest{
			Generals: &cligrpc.GeneralParams{
				MethodName: "channelTransferByAdmin",
				RequestId:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IdTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Address:    channelTransferArgs[2],
			Token:      channelTransferArgs[3],
			Amount:     channelTransferArgs[4],
		}

		By("sending transfer request")
		r, err := c.TransferByAdmin(clientCtx, transfer)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Status).To(Equal(cligrpc.TransferStatusResponse_STATUS_IN_PROCESS))

		transferStatusRequest := &cligrpc.TransferStatusRequest{
			IdTransfer: transferID,
		}

		excludeStatus := cligrpc.TransferStatusResponse_STATUS_IN_PROCESS.String()
		value, err := anypb.New(wrapperspb.String(excludeStatus))
		Expect(err).NotTo(HaveOccurred())

		transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
			Name:  "excludeStatus",
			Value: value,
		})

		ctx, cancel := context.WithTimeout(clientCtx, 120*time.Second)
		defer cancel()

		By("awaiting for channel transfer to respond")
		statusResponse, err := c.TransferStatus(ctx, transferStatusRequest)
		Expect(err).NotTo(HaveOccurred())
		Expect(statusResponse.Status).To(Equal(cligrpc.TransferStatusResponse_STATUS_COMPLETED))

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			checkResult(checkBalance("750"), nil),
			"balanceOf", user1.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelFiat,
			checkResult(checkBalance("250"), nil),
			"allowedBalanceOf", user1.AddressBase58Check)

	})

})
