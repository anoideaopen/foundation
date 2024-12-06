package basic

import (
	"encoding/json"

	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const fnMethodWithRights = "withRights"

var _ = Describe("Basic foundation Tests", func() {
	var ts client.TestSuite

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})
	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	Describe("foundation test", func() {
		var channels = []string{cmn.ChannelACL, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}

		BeforeEach(func() {
			By("start redis")
			ts.StartRedis()
		})
		BeforeEach(func() {
			ts.InitNetwork(channels, integration.DevModePort)
			ts.DeployChaincodes()
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
			user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			ts.AddUser(user)
		})

		It("check metadata in chaincode", func() {
			By("querying the chaincode from cc")
			ts.Metadata(cmn.ChannelCC, cmn.ChannelCC).CheckResultContains(`{"name":"Currency Coin","symbol":"CC","decimals":8,"underlying_asset":"US Dollars"`)

			By("querying the chaincode from fiat")
			ts.Metadata(cmn.ChannelFiat, cmn.ChannelFiat).CheckResultContains(`{"name":"FIAT","symbol":"FIAT","decimals":8,"underlying_asset":"US Dollars"`)

			By("querying the chaincode from industrial")
			ts.Metadata(cmn.ChannelIndustrial, cmn.ChannelIndustrial).CheckResultContains(`{"name":"Industrial token","symbol":"INDUSTRIAL","decimals":8,"underlying_asset":"TEST_UnderlyingAsset"`)
		})

		It("query test", func() {
			user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			ts.AddUser(user)

			By("send a request that is similar to invoke")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
				"allowedBalanceAdd", "CC", user.AddressBase58Check, "50", "add some assets").CheckBalance("Ok")

			By("let's check the allowed balance - 1")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
				"allowedBalanceOf", user.AddressBase58Check, "CC").CheckBalance("0")

			By("send an invoke that is similar to request")
			ts.NBTxInvoke(cmn.ChannelFiat, cmn.ChannelFiat, "allowedBalanceAdd", "CC", user.AddressBase58Check, "50", "add some assets").CheckErrorIsNil()

			By("let's check the allowed balance - 2")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
				"allowedBalanceOf", user.AddressBase58Check, "CC").CheckBalance("0")
		})

		Describe("transfer tests", func() {
			var (
				user1 *mocks.UserFoundation
				user2 *mocks.UserFoundation
			)

			BeforeEach(func() {
				By("add admin to acl")
				ts.AddAdminToACL()

				By("create users")
				var err error

				user1, err = mocks.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())
				user2, err = mocks.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())
			})

			It("transfer", func() {
				By("add users to acl")
				ts.AddUser(user1)
				ts.AddUser(user2)

				By("emit tokens")
				amount := "1"
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
					"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, amount).CheckErrorIsNil()

				By("emit check")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", user1.AddressBase58Check).CheckBalance(amount)

				By("get transfer fee from user1 to user2")
				req := FeeTransferRequestDTO{
					SenderAddress:    user1.AddressBase58Check,
					RecipientAddress: user2.AddressBase58Check,
					Amount:           amount,
				}
				bytes, err := json.Marshal(req)
				Expect(err).NotTo(HaveOccurred())

				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "getFeeTransfer", string(bytes)).CheckErrorEquals("fee address is not set in token config")

				By("transfer tokens from user1 to user2")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), user2.AddressBase58Check, amount, "ref transfer").CheckErrorIsNil()

				By("check balance user1")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance("0")

				By("check balance user2")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user2.AddressBase58Check).CheckBalance(amount)
			})

			It("transfer with fee", func() {
				By("add users to acl")
				user1.UserID = "1111"
				user2.UserID = "2222"

				ts.AddUser(user1)
				ts.AddUser(user2)
				ts.AddFeeSetterToACL()
				ts.AddFeeAddressSetterToACL()

				feeWallet, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())

				ts.AddUser(feeWallet)

				By("emit tokens")
				amount := "3"
				amountOne := "1"
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(), "emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, amount).CheckErrorIsNil()

				By("emit check")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", user1.AddressBase58Check).CheckBalance(amount)

				By("set fee")
				ts.TxInvokeWithSign(
					cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeSetter(),
					"setFee", "", client.NewNonceByTime().Get(), "FIAT", "1", "1", "100").CheckErrorIsNil()

				By("set fee address")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeAddressSetter(),
					"setFeeAddress", "", client.NewNonceByTime().Get(), feeWallet.AddressBase58Check).CheckErrorIsNil()

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
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "getFeeTransfer", string(bytes)).CheckResponseWithFunc(fFeeTransfer)

				By("transfer tokens from user1 to user2")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), user2.AddressBase58Check, amountOne, "ref transfer").CheckErrorIsNil()

				By("check balance user1")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", user1.AddressBase58Check).CheckBalance(amountOne)

				By("check balance user2")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", user2.AddressBase58Check).CheckBalance(amountOne)

				By("check balance feeWallet")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", feeWallet.AddressBase58Check).CheckBalance(amountOne)
			})

			It("transfer to itself to second wallet with fee is on", func() {
				By("add users to acl")
				user1.UserID = "1111"
				user2.UserID = "1111"

				ts.AddUser(user1)
				ts.AddUser(user2)
				ts.AddFeeSetterToACL()
				ts.AddFeeAddressSetterToACL()

				feeWallet, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())

				ts.AddUser(feeWallet)

				By("emit tokens")
				amount := "3"
				amountOne := "1"
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
					"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, amount).CheckErrorIsNil()

				By("emit check")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", user1.AddressBase58Check).CheckBalance(amount)

				By("set fee")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeSetter(),
					"setFee", "", client.NewNonceByTime().Get(), "FIAT", "1", "1", "100").CheckErrorIsNil()

				By("set fee address")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeAddressSetter(),
					"setFeeAddress", "", client.NewNonceByTime().Get(), feeWallet.AddressBase58Check).CheckErrorIsNil()

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
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "getFeeTransfer", string(bytes)).CheckResponseWithFunc(fFeeTransfer)

				By("transfer tokens from user1 to user2")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), user2.AddressBase58Check, amountOne, "ref transfer").CheckErrorIsNil()

				By("check balance user1")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", user1.AddressBase58Check).CheckBalance("2")

				By("check balance user2")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", user2.AddressBase58Check).CheckBalance(amountOne)

				By("check balance feeWallet")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", feeWallet.AddressBase58Check).CheckBalance("0")
			})

			It("transfer to the same wallet with fee is on", func() {
				By("add users to acl")
				ts.AddUser(user1)
				ts.AddFeeSetterToACL()
				ts.AddFeeAddressSetterToACL()

				feeWallet, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
				Expect(err).NotTo(HaveOccurred())

				ts.AddUser(feeWallet)

				By("emit tokens")
				amount := "3"
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
					"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, amount).CheckErrorIsNil()

				By("emit check")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", user1.AddressBase58Check).CheckBalance(amount)

				By("set fee")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeSetter(),
					"setFee", "", client.NewNonceByTime().Get(), "FIAT", "1", "1", "100").CheckErrorIsNil()

				By("set fee address")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.FeeAddressSetter(),
					"setFeeAddress", "", client.NewNonceByTime().Get(), feeWallet.AddressBase58Check).CheckErrorIsNil()

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
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "getFeeTransfer", string(bytes)).CheckResponseWithFunc(fFeeTransfer)

				By("NEGATIVE: transfer tokens from user1 to user2")
				ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "transfer", "",
					client.NewNonceByTime().Get(), user1.AddressBase58Check, "1", "ref transfer").CheckErrorEquals("TxTransfer: sender and recipient are same users")

				By("check balance user1")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", user1.AddressBase58Check).CheckBalance(amount)

				By("check balance feeWallet")
				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
					"balanceOf", feeWallet.AddressBase58Check).CheckBalance("0")
			})
		})

		It("accessmatrix - add and remove rights", func() {
			By("add user to acl")
			user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			ts.AddUser(user1)

			user2, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			ts.AddUser(user2)

			By("invoking industrial chaincode with user have no rights")
			ts.TxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, fnMethodWithRights, "",
				client.NewNonceByTime().Get()).CheckErrorEquals("unauthorized")

			By("add rights and check rights")
			ts.AddRights(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "issuer", "", user1)

			By("invoking industrial chaincode with acl right user")
			ts.TxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, fnMethodWithRights, "",
				client.NewNonceByTime().Get()).CheckErrorIsNil()

			By("remove rights and check rights")
			ts.RemoveRights(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "issuer", "", user1)

			By("invoking industrial chaincode with user acl rights removed")
			ts.TxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, fnMethodWithRights, "",
				client.NewNonceByTime().Get()).CheckErrorEquals("unauthorized")

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
