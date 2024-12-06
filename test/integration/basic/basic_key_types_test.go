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

var _ = Describe("Basic foundation tests with different key types", func() {
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

			By("stop redis")
			ts.StopRedis()
		})

		It("transfer", func() {
			By("create users")
			user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			user2, err := mocks.NewUserFoundation(pbfound.KeyType_secp256k1)
			Expect(err).NotTo(HaveOccurred())

			By("add users to acl")
			ts.AddUser(user1)
			ts.AddUser(user2)

			By("add admin to acl")
			ts.AddAdminToACL()

			By("emit tokens")
			amount := "1"
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, amount).CheckErrorIsNil()

			By("emit check")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(amount)

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
	})
})
