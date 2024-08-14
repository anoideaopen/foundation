package basic

import (
	"encoding/json"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Basic foundation tests with different key types", func() {
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

			By("stop redis")
			ts.StopRedis()
		})

		It("transfer", func() {
			By("create users")
			user1, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			user2, err := client.NewUserFoundation(pbfound.KeyType_secp256k1)
			Expect(err).NotTo(HaveOccurred())

			By("add users to acl")
			ts.AddUser(user1)
			ts.AddUser(user2)

			By("add admin to acl")
			ts.AddAdminToACL()

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
	})
})
