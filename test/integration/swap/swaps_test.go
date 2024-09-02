package swap

import (
	"encoding/json"

	"github.com/anoideaopen/foundation/core/types"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Swap Tests", func() {
	var (
		channels = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
		ts       client.TestSuite
	)

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})
	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	BeforeEach(func() {
		By("start redis")
		ts.StartRedis()
	})

	BeforeEach(func() {
		ts.InitNetwork(channels, integration.E2EBasePort)
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

	Describe("swap tests", func() {
		var (
			user1           *client.UserFoundation
			swapAmount      = "1"
			zeroAmount      = "0"
			defaultSwapHash = "7d4e3eec80026719639ed4dba68916eb94c7a49a053e05c8f9578fe4e5a3d7ea"
			defaultSwapKey  = "12345"
		)

		BeforeEach(func() {
			By("add admin to acl")
			ts.AddAdminToACL()

			By("add user to acl")
			var err error
			user1, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			ts.AddUser(user1)

			By("emit tokens 1000")
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, swapAmount).CheckErrorIsNil()

			By("emit check")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(swapAmount)
		})

		It("swap token from fiat to cc then swap from cc to fiat", func() {
			By("swap from fiat to cc")
			By("swap begin")
			swapBeginTxID := ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"swapBegin", "", client.NewNonceByTime().Get(),
				"FIAT", "CC", swapAmount, defaultSwapHash).TxID()
			Expect(swapBeginTxID).ToNot(BeEmpty())

			By("swap get")
			ts.SwapGet(cmn.ChannelCC, cmn.ChannelCC, client.SfnSwapGet, swapBeginTxID)

			By("check balance 1")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(zeroAmount)

			By("check allowed balance 1")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(zeroAmount)

			By("swap done")
			ts.NBTxInvoke(cmn.ChannelCC, cmn.ChannelCC, "swapDone", swapBeginTxID, defaultSwapKey).CheckErrorIsNil()

			By("check balance 2")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(zeroAmount)

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(swapAmount)

			By("swap from cc to fiat")
			By("swap begin")
			swapBeginTxID = ts.TxInvokeWithSign(cmn.ChannelCC, cmn.ChannelCC, user1,
				"swapBegin", "", client.NewNonceByTime().Get(),
				"FIAT", "FIAT", swapAmount, defaultSwapHash).TxID()
			Expect(swapBeginTxID).ToNot(BeEmpty())

			By("swap get")
			ts.SwapGet(cmn.ChannelFiat, cmn.ChannelFiat, client.SfnSwapGet, swapBeginTxID)

			By("check balance 1")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(zeroAmount)

			By("check allowed balance 1")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(zeroAmount)

			By("swap done")
			ts.NBTxInvoke(cmn.ChannelFiat, cmn.ChannelFiat, "swapDone", swapBeginTxID, defaultSwapKey).CheckErrorIsNil()

			By("check balance 2")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(swapAmount)

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(zeroAmount)
		})

		It("swap token from fiat to cc and swap cancel", func() {
			By("swap from fiat to cc")
			By("swap begin")
			swapBeginTxID := ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"swapBegin", "", client.NewNonceByTime().Get(),
				"FIAT", "CC", swapAmount, defaultSwapHash).TxID()
			Expect(swapBeginTxID).ToNot(BeEmpty())

			By("swap get")
			ts.SwapGet(cmn.ChannelCC, cmn.ChannelCC, client.SfnSwapGet, swapBeginTxID)

			By("check balance 1")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(zeroAmount)

			By("check allowed balance 1")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(zeroAmount)

			By("swap cancel on channel cc")
			ts.TxInvokeWithSign(cmn.ChannelCC, cmn.ChannelCC, user1,
				"swapCancel", "", client.NewNonceByTime().Get(), swapBeginTxID).CheckErrorIsNil()

			By("swap cancel on channel fiat")
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"swapCancel", "", client.NewNonceByTime().Get(), swapBeginTxID).CheckErrorIsNil()

			By("check balance 2")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(swapAmount)

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(zeroAmount)
		})
	})

	Describe("multiswap tests", func() {
		var (
			user1           *client.UserFoundation
			swapAmount      = "1"
			zeroAmount      = "0"
			defaultSwapHash = "7d4e3eec80026719639ed4dba68916eb94c7a49a053e05c8f9578fe4e5a3d7ea"
			defaultSwapKey  = "12345"
		)

		BeforeEach(func() {
			By("add admin to acl")
			ts.AddAdminToACL()

			By("add user to acl")
			var err error
			user1, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			ts.AddUser(user1)

			By("emit tokens 1000")
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, swapAmount).CheckErrorIsNil()

			By("emit check")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(swapAmount)
		})

		It("multiswap token from fiat to cc then multiswap from cc to fiat", func() {
			By("multiswap from fiat to cc")
			By("multiswap begin")
			assets, err := json.Marshal(types.MultiSwapAssets{
				Assets: []*types.MultiSwapAsset{
					{
						Group:  "FIAT",
						Amount: swapAmount,
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			swapBeginTxID := ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"multiSwapBegin", "", client.NewNonceByTime().Get(),
				"FIAT", string(assets), "CC", defaultSwapHash).TxID()
			Expect(swapBeginTxID).ToNot(BeEmpty())

			By("multiswap get 1")
			ts.SwapGet(cmn.ChannelCC, cmn.ChannelCC, client.SfnMultiSwapGet, swapBeginTxID)

			By("check balance 1")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(zeroAmount)

			By("check allowed balance 1")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(zeroAmount)

			By("multiswap done")
			ts.NBTxInvoke(cmn.ChannelCC, cmn.ChannelCC, "multiSwapDone", swapBeginTxID, defaultSwapKey).CheckErrorIsNil()

			By("check balance 2")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(zeroAmount)

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(swapAmount)

			By("multiswap from cc to fiat")
			By("multiswap begin")
			swapBeginTxID = ts.TxInvokeWithSign(
				cmn.ChannelCC, cmn.ChannelCC, user1,
				"multiSwapBegin", "", client.NewNonceByTime().Get(),
				"FIAT", string(assets), "FIAT", defaultSwapHash).TxID()
			Expect(swapBeginTxID).ToNot(BeEmpty())

			By("multiswap get 2")
			ts.SwapGet(cmn.ChannelFiat, cmn.ChannelFiat, client.SfnMultiSwapGet, swapBeginTxID)

			By("check balance 3")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(zeroAmount)

			By("check allowed balance 3")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(zeroAmount)

			By("multiswap done")
			ts.NBTxInvoke(cmn.ChannelFiat, cmn.ChannelFiat, "multiSwapDone", swapBeginTxID, defaultSwapKey).CheckErrorIsNil()

			By("check balance 4")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(swapAmount)

			By("check allowed balance 4")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(zeroAmount)
		})
	})
})
