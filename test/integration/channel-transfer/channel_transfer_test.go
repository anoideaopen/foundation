package channel_transfer

import (
	"context"
	"strconv"

	cligrpc "github.com/anoideaopen/channel-transfer/proto"
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

var _ = Describe("Channel transfer foundation Tests", func() {
	var ts client.TestSuite

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})
	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	var (
		channels     = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
		user         *client.UserFoundation
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
		ts.InitNetwork(channels, integration.GossipBasePort)
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

	BeforeEach(func() {
		network = ts.Network()
		networkFound = ts.NetworkFound()

		By("add admin to acl")
		ts.AddAdminToACL()

		By("add user to acl")
		var err error
		user, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user)

		By("emit tokens")
		emitAmount := "1000"
		ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
			"emit", "", client.NewNonceByTime().Get(), user.AddressBase58Check, emitAmount).CheckErrorIsNil()

		By("emit check")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

	})

	It("transfer by admin test", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)

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
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, "CC", user.AddressBase58Check, "FIAT", "250"}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{"channelTransferByAdmin", requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := ts.Admin().Sign(signArgs...)
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
		r, err := apiClient.TransferByAdmin(clientCtx, transfer)
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
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user.AddressBase58Check).CheckBalance("750")

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user.AddressBase58Check, "FIAT").CheckBalance("250")
	})

	It("transfer by customer test", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)

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
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, "CC", "FIAT", "250"}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{"channelTransferByCustomer", requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := user.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transfer := &cligrpc.TransferBeginCustomerRequest{
			Generals: &cligrpc.GeneralParams{
				MethodName: "channelTransferByCustomer",
				RequestId:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IdTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Token:      channelTransferArgs[2],
			Amount:     channelTransferArgs[3],
		}

		By("sending transfer request")
		r, err := apiClient.TransferByCustomer(clientCtx, transfer)
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
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user.AddressBase58Check).CheckBalance("750")

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user.AddressBase58Check, "FIAT").CheckBalance("250")
	})

	It("transfer status with wrong transfer id test", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransferGRPCAddress()

		var err error

		conn, err = grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(transportCredentials))
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}()

		By("creating channel transfer API client")
		apiClient = cligrpc.NewAPIClient(conn)

		By("requesting status of transfer with id = 1")
		transferStatusRequest := &cligrpc.TransferStatusRequest{
			IdTransfer: "1",
		}
		_, err = apiClient.TransferStatus(clientCtx, transferStatusRequest)
		Expect(err).To(MatchError(ContainSubstring("object not found")))
	})

	It("transfer status filter test", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)

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
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, "CC", "FIAT", "250"}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{"channelTransferByCustomer", requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := user.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transfer := &cligrpc.TransferBeginCustomerRequest{
			Generals: &cligrpc.GeneralParams{
				MethodName: "channelTransferByCustomer",
				RequestId:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IdTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Token:      channelTransferArgs[2],
			Amount:     channelTransferArgs[3],
		}

		By("sending transfer request")
		r, err := apiClient.TransferByCustomer(clientCtx, transfer)
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

	})

	It("transfer wrong STATUS_CANCELLED filter test", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)

		var err error

		conn, err = grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(transportCredentials))
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}()

		By("creating channel transfer API client")
		apiClient = cligrpc.NewAPIClient(conn)

		By("requesting status of transfer with id = 1")
		transferStatusRequest := &cligrpc.TransferStatusRequest{
			IdTransfer: "1",
		}

		By("setting STATUS_CANCELLED filter")
		excludeStatus := cligrpc.TransferStatusResponse_STATUS_CANCELED.String()
		value, err := anypb.New(wrapperspb.String(excludeStatus))
		Expect(err).NotTo(HaveOccurred())

		transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
			Name:  "excludeStatus",
			Value: value,
		})

		By("checking status")
		_, err = apiClient.TransferStatus(clientCtx, transferStatusRequest)
		Expect(err).To(MatchError(ContainSubstring("exclude status not valid")))
	})

	It("transfer wrong STATUS_COMPLETED filter test", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)

		var err error

		conn, err = grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(transportCredentials))
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}()

		By("creating channel transfer API client")
		apiClient = cligrpc.NewAPIClient(conn)

		By("requesting status of transfer with id = 1")
		transferStatusRequest := &cligrpc.TransferStatusRequest{
			IdTransfer: "1",
		}

		By("setting STATUS_COMPLETED filter")
		excludeStatus := cligrpc.TransferStatusResponse_STATUS_COMPLETED.String()
		value, err := anypb.New(wrapperspb.String(excludeStatus))
		Expect(err).NotTo(HaveOccurred())

		transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
			Name:  "excludeStatus",
			Value: value,
		})

		By("checking status")
		_, err = apiClient.TransferStatus(clientCtx, transferStatusRequest)
		Expect(err).To(MatchError(ContainSubstring("exclude status not valid")))
	})

	It("transfer wrong STATUS_ERROR filter test", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)

		var err error

		conn, err = grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(transportCredentials))
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}()

		By("creating channel transfer API client")
		apiClient = cligrpc.NewAPIClient(conn)

		By("requesting status of transfer with id = 1")
		transferStatusRequest := &cligrpc.TransferStatusRequest{
			IdTransfer: "1",
		}

		By("setting STATUS_ERROR filter")
		excludeStatus := cligrpc.TransferStatusResponse_STATUS_ERROR.String()
		value, err := anypb.New(wrapperspb.String(excludeStatus))
		Expect(err).NotTo(HaveOccurred())

		transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
			Name:  "excludeStatus",
			Value: value,
		})

		By("checking status")
		_, err = apiClient.TransferStatus(clientCtx, transferStatusRequest)
		Expect(err).To(MatchError(ContainSubstring("exclude status not valid")))
	})

	It("transfer undefined status filter test", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)

		var err error

		conn, err = grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(transportCredentials))
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}()

		By("creating channel transfer API client")
		apiClient = cligrpc.NewAPIClient(conn)

		By("requesting status of transfer with id = 1")
		transferStatusRequest := &cligrpc.TransferStatusRequest{
			IdTransfer: "1",
		}

		By("setting not defined status filter")
		excludeStatus := "9999999"
		value, err := anypb.New(wrapperspb.String(excludeStatus))
		Expect(err).NotTo(HaveOccurred())

		transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
			Name:  "excludeStatus",
			Value: value,
		})

		By("checking status")
		_, err = apiClient.TransferStatus(clientCtx, transferStatusRequest)
		Expect(err).To(MatchError(ContainSubstring("exclude status not found")))
	})
})
