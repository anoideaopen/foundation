package channel_transfer_multi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	cligrpc "github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ = Describe("Channel transfer multi-transfers foundation Tests", func() {
	var (
		ts *client.FoundationTestSuite
	)

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})
	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	var (
		channels     = []string{cmn.ChannelACL, cmn.ChannelCC, cmn.ChannelIndustrial}
		user         *mocks.UserFoundation
		clientCtx    context.Context
		apiClient    cligrpc.APIClient
		conn         *grpc.ClientConn
		network      *nwo.Network
		networkFound *cmn.NetworkFoundation
	)

	BeforeEach(func() {
		By("start redis")
		ts.StartRedis()
	})

	BeforeEach(func() {
		ts.InitNetwork(channels, integration.KafkaBasePort)
		ts.DeployChaincodes()
	})

	BeforeEach(func() {
		By("start robot")
		ts.StartRobot()
		By("start channel transfer")
		ts.StartChannelTransfer()
	})

	AfterEach(func() {
		By("stop robot")
		ts.StopRobot()
		By("stop channel transfer")
		ts.StopChannelTransfer()
		By("stop redis")
		ts.StopRedis()
	})

	var (
		initialBalances            []*cligrpc.TransferItem
		transferItems              []*cligrpc.TransferItem
		expectedIndustrialBalances []*cligrpc.TransferItem
	)

	BeforeEach(func() {
		network = ts.Network
		networkFound = ts.NetworkFound

		By("add admin to acl")
		ts.AddAdminToACL()

		By("add user to acl")
		var err error
		user, err = mocks.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user)

		By("emit tokens")
		ts.NBTxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial,
			ts.Admin(), "initialize", "", client.NewNonceByTime().Get())

		initialBalances = []*cligrpc.TransferItem{
			{
				Token:  "INDUSTRIAL_202009",
				Amount: "10000000000000",
			},
			{
				Token:  "INDUSTRIAL_202010",
				Amount: "100000000000000",
			},
			{
				Token:  "INDUSTRIAL_202011",
				Amount: "200000000000000",
			},
			{
				Token:  "INDUSTRIAL_202012",
				Amount: "50000000000000",
			},
		}

		transferItems = []*cligrpc.TransferItem{
			{
				Token:  "INDUSTRIAL_202009",
				Amount: "1000000000000",
			},
			{
				Token:  "INDUSTRIAL_202010",
				Amount: "10000000000000",
			},
			{
				Token:  "INDUSTRIAL_202011",
				Amount: "20000000000000",
			},
			{
				Token:  "INDUSTRIAL_202012",
				Amount: "5000000000000",
			},
		}

		expectedIndustrialBalances = []*cligrpc.TransferItem{
			{
				Token:  "INDUSTRIAL_202009",
				Amount: "9000000000000",
			},
			{
				Token:  "INDUSTRIAL_202010",
				Amount: "90000000000000",
			},
			{
				Token:  "INDUSTRIAL_202011",
				Amount: "180000000000000",
			},
			{
				Token:  "INDUSTRIAL_202012",
				Amount: "45000000000000",
			},
		}

		for _, initial := range initialBalances {
			group := strings.Split(initial.GetToken(), "_")[1]

			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				"industrialBalanceOf", ts.Admin().AddressBase58Check).
				CheckIndustrialBalance(group, initial.Amount)

			ts.TxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, ts.Admin(),
				"transferIndustrial", "", client.NewNonceByTime().Get(),
				user.AddressBase58Check, group, initial.Amount, "transfer industrial tokens")

			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				"industrialBalanceOf", user.AddressBase58Check).
				CheckIndustrialBalance(group, initial.Amount)
		}
	})

	It("multi transfer by admin test", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := net.JoinHostPort(networkFound.ChannelTransfer.HostAddress, fmt.Sprintf("%d", networkFound.ChannelTransfer.Ports[cmn.GrpcPort]))

		var err error

		conn, err = grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(transportCredentials))
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}()

		By("creating channel transfer API client")
		apiClient = cligrpc.NewAPIClient(conn)

		By("creating channel transfer request")
		items, err := json.Marshal(transferItems)
		Expect(err).NotTo(HaveOccurred())

		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, "CC", user.AddressBase58Check, string(items)}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{"channelMultiTransferByAdmin", requestID, cmn.ChannelIndustrial, cmn.ChannelIndustrial}, channelTransferArgs...), nonce)
		publicKey, sign, err := ts.Admin().Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transfer := &cligrpc.MultiTransferBeginAdminRequest{
			Generals: &cligrpc.GeneralParams{
				MethodName: "channelMultiTransferByAdmin",
				RequestId:  requestID,
				Chaincode:  cmn.ChannelIndustrial,
				Channel:    cmn.ChannelIndustrial,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IdTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Address:    channelTransferArgs[2],
			Items:      transferItems,
		}

		By("sending multi transfer request")
		r, err := apiClient.MultiTransferByAdmin(clientCtx, transfer)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Status).To(Equal(cligrpc.TransferStatusResponse_STATUS_IN_PROCESS))

		By("checking multi transfer status")
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

		ctx, cancel := context.WithTimeout(clientCtx, network.EventuallyTimeout*2)
		defer cancel()

		By("awaiting for channel transfer to respond")
		statusResponse, err := apiClient.TransferStatus(ctx, transferStatusRequest)
		Expect(err).NotTo(HaveOccurred())
		Expect(statusResponse.Status).To(Equal(cligrpc.TransferStatusResponse_STATUS_COMPLETED))

		By("checking result balances")
		for i, expected := range expectedIndustrialBalances {
			group := strings.Split(expected.GetToken(), "_")[1]

			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				"industrialBalanceOf", user.AddressBase58Check).
				CheckIndustrialBalance(group, expected.GetAmount())

			ts.Query(cmn.ChannelCC, cmn.ChannelCC,
				"allowedBalanceOf", user.AddressBase58Check, transferItems[i].GetToken()).
				CheckBalance(transferItems[i].GetAmount())
		}
	})

	It("multi transfer by customer test", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := net.JoinHostPort(networkFound.ChannelTransfer.HostAddress, fmt.Sprintf("%d", networkFound.ChannelTransfer.Ports[cmn.GrpcPort]))

		var err error

		conn, err = grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(transportCredentials))
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}()

		By("creating channel transfer API client")
		apiClient = cligrpc.NewAPIClient(conn)

		By("creating channel transfer request")
		items, err := json.Marshal(transferItems)
		Expect(err).NotTo(HaveOccurred())

		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, "CC", string(items)}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{"channelMultiTransferByCustomer", requestID, cmn.ChannelIndustrial, cmn.ChannelIndustrial}, channelTransferArgs...), nonce)
		publicKey, sign, err := user.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transfer := &cligrpc.MultiTransferBeginCustomerRequest{
			Generals: &cligrpc.GeneralParams{
				MethodName: "channelMultiTransferByCustomer",
				RequestId:  requestID,
				Chaincode:  cmn.ChannelIndustrial,
				Channel:    cmn.ChannelIndustrial,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IdTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Items:      transferItems,
		}

		By("sending transfer request")
		r, err := apiClient.MultiTransferByCustomer(clientCtx, transfer)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Status).To(Equal(cligrpc.TransferStatusResponse_STATUS_IN_PROCESS))

		By("checking transfer status")
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

		ctx, cancel := context.WithTimeout(clientCtx, network.EventuallyTimeout*2)
		defer cancel()

		By("awaiting for channel transfer to respond")
		statusResponse, err := apiClient.TransferStatus(ctx, transferStatusRequest)
		Expect(err).NotTo(HaveOccurred())
		Expect(statusResponse.Status).To(Equal(cligrpc.TransferStatusResponse_STATUS_COMPLETED))

		By("checking result balances")
		for i, expected := range expectedIndustrialBalances {
			group := strings.Split(expected.GetToken(), "_")[1]

			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				"industrialBalanceOf", user.AddressBase58Check).
				CheckIndustrialBalance(group, expected.GetAmount())

			ts.Query(cmn.ChannelCC, cmn.ChannelCC,
				"allowedBalanceOf", user.AddressBase58Check, transferItems[i].GetToken()).
				CheckBalance(transferItems[i].GetAmount())
		}
	})
})
