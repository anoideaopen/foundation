package custom_acl_channel_test

import (
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	buildServer *nwo.BuildServer
	components  *nwo.Components
)

var _ = Describe("Basic foundation tests with different key types", func() {
	var ts client.TestSuite

	BeforeEach(func() {
		ts = client.NewTestSuite(components, client.WithACLChannelName("acl2"))
	})

	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	Describe("foundation test", func() {
		channels := []string{cmn.ChannelACL, cmn.ChannelFiat}

		BeforeEach(func() {
			By("start redis")
			ts.StartRedis()
		})
		BeforeEach(func() {
			ts.InitNetwork(channels, integration.DevModePort)
			ts.DeployChaincodes()
		})

		It("emit using unusual acl channel name", func() {
			By("create users")
			user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			By("add users to acl")
			ts.AddUser(user1)

			By("add admin to acl")
			ts.AddAdminToACL()

			By("emit tokens")
			amount := "1"
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, amount).CheckErrorIsNil()

			By("emit check")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(amount)
		})
	})
})
