package basic

import (
	"encoding/json"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

const fnMethodWithRights = "withRights"

var _ = Describe("Basic foundation Tests", func() {
	var (
		ts       client.TestSuite
		channels = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
	)

	BeforeEach(func() {
		ts = client.NewTestSuite(components, channels)
	})
	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	Describe("foundation test", func() {
		BeforeEach(func() {
			By("start redis")
			ts.StartRedis()
		})
		BeforeEach(func() {
			ts.InitNetwork(integration.DevModePort)
			ts.DeployChannels()
		})
		BeforeEach(func() {
			By("start robot")
			ts.StartRobot()
		})
		AfterEach(func() {
			By("stop robot")
			ts.StopRobot()
		})
		AfterEach(func() {
			By("stop redis")
			ts.StopRedis()
		})

		It("add user", func() {
			user, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			ts.AddUser(user)
		})

		It("check metadata in chaincode", func() {
			network := ts.Network()
			peer := ts.Peer()
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
			ts.AddUser(user)

			By("send a request that is similar to invoke")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
				client.CheckResult(client.CheckBalance("Ok"), nil),
				"allowedBalanceAdd", "CC", user.AddressBase58Check, "50", "add some assets")

			By("let's check the allowed balance - 1")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
				client.CheckResult(client.CheckBalance("0"), nil),
				"allowedBalanceOf", user.AddressBase58Check, "CC")

			By("send an invoke that is similar to request")
			ts.NBTxInvoke(cmn.ChannelFiat, cmn.ChannelFiat, nil, "allowedBalanceAdd", "CC", user.AddressBase58Check, "50", "add some assets")

			By("let's check the allowed balance - 2")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
				client.CheckResult(client.CheckBalance("0"), nil),
				"allowedBalanceOf", user.AddressBase58Check, "CC")
		})

		Describe("transfer tests", func() {
			var (
				user1 *client.UserFoundation
				user2 *client.UserFoundation
			)

			BeforeEach(func() {
				By("add admin to acl")
				ts.AddAdminToACL()

				By("create users")
				var err error

				user1, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())
				user2, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())
			})

			It("transfer", func() {
				By("add users to acl")
				ts.AddUser(user1)
				ts.AddUser(user2)

				By("emit tokens")
				amount := "1"
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
					"emit", "", client.NewNonceByTime().Get(), nil, user1.AddressBase58Check, amount)

				By("emit check")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance(amount), nil),
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
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, client.CheckResult(nil, fErr),
					"getFeeTransfer", string(bytes))

				By("transfer tokens from user1 to user2")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), nil, user2.AddressBase58Check, amount, "ref transfer")

				By("check balance user1")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance("0"), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance user2")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance(amount), nil),
					"balanceOf", user2.AddressBase58Check)
			})

			It("transfer with fee", func() {
				By("add users to acl")
				user1.UserID = "1111"
				user2.UserID = "2222"

				ts.AddUser(user1)
				ts.AddUser(user2)
				ts.AddFeeSetterToACL()
				ts.AddFeeAddressSetterToACL()

				feeWallet, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())

				ts.AddUser(feeWallet)

				By("emit tokens")
				amount := "3"
				amountOne := "1"
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(), "emit", "", client.NewNonceByTime().Get(), nil, user1.AddressBase58Check, amount)

				By("emit check")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("set fee")
				ts.TxInvokeWithSign(
					cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeSetter(),
					"setFee", "", client.NewNonceByTime().Get(), nil, "FIAT", "1", "1", "100")

				By("set fee address")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeAddressSetter(),
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
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, client.CheckResult(fFeeTransfer, nil),
					"getFeeTransfer", string(bytes))

				By("transfer tokens from user1 to user2")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), nil, user2.AddressBase58Check, amountOne, "ref transfer")

				By("check balance user1")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance(amountOne), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance user2")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance(amountOne), nil),
					"balanceOf", user2.AddressBase58Check)

				By("check balance feeWallet")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance(amountOne), nil),
					"balanceOf", feeWallet.AddressBase58Check)
			})

			It("transfer to itself to second wallet with fee is on", func() {
				By("add users to acl")
				user1.UserID = "1111"
				user2.UserID = "1111"

				ts.AddUser(user1)
				ts.AddUser(user2)
				ts.AddFeeSetterToACL()
				ts.AddFeeAddressSetterToACL()

				feeWallet, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())

				ts.AddUser(feeWallet)

				By("emit tokens")
				amount := "3"
				amountOne := "1"
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
					"emit", "", client.NewNonceByTime().Get(), nil, user1.AddressBase58Check, amount)

				By("emit check")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("set fee")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeSetter(),
					"setFee", "", client.NewNonceByTime().Get(), nil, "FIAT", "1", "1", "100")

				By("set fee address")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeAddressSetter(),
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
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, client.CheckResult(fFeeTransfer, nil),
					"getFeeTransfer", string(bytes))

				By("transfer tokens from user1 to user2")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), nil, user2.AddressBase58Check, amountOne, "ref transfer")

				By("check balance user1")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance("2"), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance user2")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance(amountOne), nil),
					"balanceOf", user2.AddressBase58Check)

				By("check balance feeWallet")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance("0"), nil),
					"balanceOf", feeWallet.AddressBase58Check)
			})

			It("transfer to the same wallet with fee is on", func() {
				By("add users to acl")
				ts.AddUser(user1)
				ts.AddFeeSetterToACL()
				ts.AddFeeAddressSetterToACL()

				feeWallet, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())

				ts.AddUser(feeWallet)

				By("emit tokens")
				amount := "3"
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
					"emit", "", client.NewNonceByTime().Get(), nil, user1.AddressBase58Check, amount)

				By("emit check")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("set fee")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeSetter(),
					"setFee", "", client.NewNonceByTime().Get(), nil, "FIAT", "1", "1", "100")

				By("set fee address")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeAddressSetter(),
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
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, client.CheckResult(fFeeTransfer, nil),
					"getFeeTransfer", string(bytes))

				By("NEGATIVE: transfer tokens from user1 to user2")
				ts.TxInvokeWithSign(
					cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), client.CheckResult(nil, client.CheckTxResponseResult("TxTransfer: sender and recipient are same users")), user1.AddressBase58Check, "1", "ref transfer")

				By("check balance user1")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance(amount), nil),
					"balanceOf", user1.AddressBase58Check)

				By("check balance feeWallet")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					client.CheckResult(client.CheckBalance("0"), nil),
					"balanceOf", feeWallet.AddressBase58Check)
			})
		})

		It("accessmatrix - add and remove rights", func() {
			By("add user to acl")
			user1, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			ts.AddUser(user1)

			user2, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			ts.AddUser(user2)

			By("invoking industrial chaincode with user have no rights")
			ts.TxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, fnMethodWithRights, "",
				client.NewNonceByTime().Get(), client.CheckResult(nil, client.CheckTxResponseResult("unauthorized")))

			By("add rights and check rights")
			ts.AddRights(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "issuer", "", user1)

			By("invoking industrial chaincode with acl right user")
			ts.TxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, fnMethodWithRights, "",
				client.NewNonceByTime().Get(), nil)

			By("remove rights and check rights")
			ts.RemoveRights(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "issuer", "", user1)

			By("invoking industrial chaincode with user acl rights removed")
			ts.TxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, fnMethodWithRights, "",
				client.NewNonceByTime().Get(), client.CheckResult(nil, client.CheckTxResponseResult("unauthorized")))

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
