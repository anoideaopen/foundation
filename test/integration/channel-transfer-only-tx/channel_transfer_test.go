package channel_transfer_only_tx

import (
	"encoding/json"
	"strings"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types/big"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Channel transfer only tx foundation Tests", func() {
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
		ts.InitNetwork(channels, integration.GatewayBasePort)
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
		ts.StartRedis()
	})

	Describe("channel transfer test", func() {
		var (
			user1                *client.UserFoundation
			transferAmount       = "450"
			balanceAfterTransfer = "550"
			emitAmount           = "1000"
			id                   string
			id2                  string
			transferItems        []core.TransferItem
		)

		BeforeEach(func() {
			By("add admin to acl")
			ts.AddAdminToACL()

			By("add user to acl")
			var err error
			user1, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			ts.AddUser(user1)

			id = uuid.NewString()
			id2 = uuid.NewString()

			By("emit tokens 1000")
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"emit", "", client.NewNonceByTime().Get(), user1.AddressBase58Check, emitAmount).CheckErrorIsNil()

			By("emit check")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(emitAmount)

			By("initialize industrial")
			ts.NBTxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				ts.Admin(), "initialize", "", client.NewNonceByTime().Get()).CheckErrorIsNil()

			transferItems = []core.TransferItem{
				{
					Token:  "INDUSTRIAL_202009",
					Amount: big.NewInt(10000000000000),
				},
				{
					Token:  "INDUSTRIAL_202010",
					Amount: big.NewInt(100000000000000),
				},
				{
					Token:  "INDUSTRIAL_202011",
					Amount: big.NewInt(200000000000000),
				},
				{
					Token:  "INDUSTRIAL_202012",
					Amount: big.NewInt(50000000000000),
				},
			}

			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					"industrialBalanceOf", ts.Admin().AddressBase58Check).CheckIndustrialBalance(group, item.Amount.String())
				ts.TxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					ts.Admin(), "transferIndustrial", "", client.NewNonceByTime().Get(),
					user1.AddressBase58Check, group, item.Amount.String(), "comment").CheckErrorIsNil()
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					"industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, item.Amount.String())
			}
		})

		It("by customer success", func() {
			By("FORWARD")

			By("channel transfer by customer forward")
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), id, "CC", "FIAT", transferAmount).CheckErrorIsNil()

			By("check balance after transfer")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
				"balanceOf", user1.AddressBase58Check).CheckBalance(balanceAfterTransfer)

			By("get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)
				return ""
			}
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferFrom", id).CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "createCCTransferTo", from).CheckErrorIsNil()

			By("check allowed balance 1")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(transferAmount)

			By("channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferTo", id).CheckResponseWithFunc(fChTrTo)

			By("commit cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "commitCCTransferFrom", id).CheckErrorIsNil()

			By("delete cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "removeCCTransferTo", id).CheckErrorIsNil()

			By("delete cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferFrom", id).CheckErrorIsNil()

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(transferAmount)

			By("check fiat balance")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(balanceAfterTransfer)

			By("BACKWARD")

			By("channel transfer by customer backward")
			ts.TxInvokeWithSign(cmn.ChannelCC, cmn.ChannelCC, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), id2, "FIAT", "FIAT", transferAmount).CheckErrorIsNil()

			By("check allowed balance after transfer")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance("0")

			By("get channel transfer from")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferFrom", id2).CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "createCCTransferTo", from).CheckErrorIsNil()

			By("check fiat balance 1")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(emitAmount)

			By("channel transfer to")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferTo", id2).CheckResponseWithFunc(fChTrTo)

			By("commit cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2).CheckErrorIsNil()

			By("delete cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "removeCCTransferTo", id2).CheckErrorIsNil()

			By("delete cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2).CheckErrorIsNil()

			By("check allowed balance")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance("0")

			By("check fiat balance 2")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(emitAmount)
		})

		It("channel transfer by admin success", func() {
			By("FORWARD")

			By("channel transfer forward")
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(), "channelTransferByAdmin", "",
				client.NewNonceByTime().Get(), id, "CC", user1.AddressBase58Check, "FIAT", transferAmount).CheckErrorIsNil()

			By("check balance after transfer")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(balanceAfterTransfer)

			By("get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)

				return ""
			}
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferFrom", id).CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "createCCTransferTo", from).CheckErrorIsNil()

			By("check allowed balance 1")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(transferAmount)

			By("channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferTo", id).CheckResponseWithFunc(fChTrTo)

			By("commit cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "commitCCTransferFrom", id).CheckErrorIsNil()

			By("delete cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "removeCCTransferTo", id).CheckErrorIsNil()

			By("delete cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferFrom", id).CheckErrorIsNil()

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(transferAmount)

			By("check fiat balance")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(balanceAfterTransfer)

			By("BACKWARD")

			By("channel transfer by admin backward")
			ts.TxInvokeWithSign(cmn.ChannelCC, cmn.ChannelCC, ts.Admin(), "channelTransferByAdmin", "",
				client.NewNonceByTime().Get(), id2, "FIAT", user1.AddressBase58Check, "FIAT", transferAmount).CheckErrorIsNil()

			By("check allowed balance after transfer")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance("0")

			By("get channel transfer from")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferFrom", id2).CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "createCCTransferTo", from).CheckErrorIsNil()

			By("check fiat balance 1")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(emitAmount)

			By("channel transfer to")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferTo", id2).CheckResponseWithFunc(fChTrTo)

			By("commit cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2).CheckErrorIsNil()

			By("delete cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "removeCCTransferTo", id2).CheckErrorIsNil()

			By("delete cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2).CheckErrorIsNil()

			By("check allowed balance")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance("0")

			By("check fiat balance 2")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(emitAmount)
		})

		It("cancel forward success", func() {
			By("cancel channel transfer forward")
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), id, "CC", "FIAT", transferAmount).CheckErrorIsNil()

			By("check balance after transfer")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(balanceAfterTransfer)

			By("get channel transfer from")
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferFrom", id).CheckResponseWithFunc(fChTrFrom)

			By("cancel cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "cancelCCTransferFrom", id).CheckErrorIsNil()

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance("0")

			By("check fiat balance")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(emitAmount)
		})

		It("cancel backward success", func() {
			By("FORWARD")

			By("channel transfer by customer forward")
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), id, "CC", "FIAT", transferAmount).CheckErrorIsNil()

			By("check balance after transfer")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(balanceAfterTransfer)

			By("get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)
				return ""
			}
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferFrom", id).CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "createCCTransferTo", from).CheckErrorIsNil()

			By("check allowed balance 1")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(transferAmount)

			By("channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferTo", id).CheckResponseWithFunc(fChTrTo)

			By("commit cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "commitCCTransferFrom", id).CheckErrorIsNil()

			By("delete cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "removeCCTransferTo", id).CheckErrorIsNil()

			By("delete cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferFrom", id).CheckErrorIsNil()

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(transferAmount)

			By("check fiat balance")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(balanceAfterTransfer)

			By("BACKWARD")

			By("channel transfer by customer backward")
			ts.TxInvokeWithSign(cmn.ChannelCC, cmn.ChannelCC, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), id2, "FIAT", "FIAT", transferAmount).CheckErrorIsNil()

			By("check allowed balance after transfer")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance("0")

			By("get channel transfer from")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferFrom", id2).CheckResponseWithFunc(fChTrTo)

			By("cancel cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "cancelCCTransferFrom", id2).CheckErrorIsNil()

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").CheckBalance(transferAmount)

			By("check fiat balance")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance(balanceAfterTransfer)
		})

		It("query all transfers from", func() {
			transferAmount = "100"
			ids := make(map[string]struct{})

			By("channel transfer by customer forward1")
			id = uuid.NewString()
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), id, "CC", "FIAT", transferAmount).CheckErrorIsNil()
			ids[id] = struct{}{}

			By("channel transfer by customer forward2")
			id = uuid.NewString()
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), id, "CC", "FIAT", transferAmount).CheckErrorIsNil()
			ids[id] = struct{}{}

			By("channel transfer by customer forward3")
			id = uuid.NewString()
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), id, "CC", "FIAT", transferAmount).CheckErrorIsNil()
			ids[id] = struct{}{}

			By("channel transfer by customer forward4")
			id = uuid.NewString()
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), id, "CC", "FIAT", transferAmount).CheckErrorIsNil()
			ids[id] = struct{}{}

			By("channel transfer by customer forward5")
			id = uuid.NewString()
			ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1, "channelTransferByCustomer", "",
				client.NewNonceByTime().Get(), id, "CC", "FIAT", transferAmount).CheckErrorIsNil()
			ids[id] = struct{}{}

			By("check balance after transfer")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance("500")

			bookmark := ""
			By("checking size")
			fSize := func(out []byte) string {
				resp := pbfound.CCTransfers{}
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Bookmark).ToNot(BeEmpty())
				Expect(resp.Ccts).To(HaveLen(2))

				return ""
			}
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransfersFrom", "2", bookmark).CheckResponseWithFunc(fSize)

			By("checking size")
			bookmark = ""
			fCheckIds := func(out []byte) string {
				resp := pbfound.CCTransfers{}
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Bookmark).To(BeEmpty())
				Expect(resp.Ccts).To(HaveLen(5))
				for _, cct := range resp.Ccts {
					Expect(ids).Should(HaveKey(cct.Id))
				}

				return ""
			}
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransfersFrom", "1000", bookmark).CheckResponseWithFunc(fCheckIds)

			count := 0
			bookmark = ""
			for {
				fCheckBookmark := func(out []byte) string {
					resp := pbfound.CCTransfers{}
					err := json.Unmarshal(out, &resp)
					Expect(err).NotTo(HaveOccurred())
					bookmark = resp.Bookmark
					return ""
				}

				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransfersFrom", "2", bookmark).CheckResponseWithFunc(fCheckBookmark)

				if bookmark == "" {
					Expect(count).To(Equal(2))
					break
				}

				count++
			}
		})

		It("multi transfer by customer success", func() {
			By("FORWARD")

			By("FORWARD. channel transfer by customer forward")
			forwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			ts.TxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1, "channelMultiTransferByCustomer", "",
				client.NewNonceByTime().Get(), id, "CC", string(forwardItemsJSON)).CheckErrorIsNil()

			By("FORWARD. check industrial balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial,
					"industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, "0")
			}

			By("FORWARD. check cc allowed balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance("0")
			}

			By("FORWARD. get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)
				return ""
			}
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "channelTransferFrom", id).CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("FORWARD. create cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "createCCTransferTo", from).CheckErrorIsNil()

			By("FORWARD. check industrial balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, "0")
			}

			By("FORWARD. check cc allowed balance after createCCTransferTo")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance(item.Amount.String())
			}

			By("FORWARD. channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferTo", id).CheckResponseWithFunc(fChTrTo)

			By("FORWARD. commit cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "commitCCTransferFrom", id).CheckErrorIsNil()

			By("FORWARD. delete cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "removeCCTransferTo", id).CheckErrorIsNil()

			By("FORWARD. delete cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "deleteCCTransferFrom", id).CheckErrorIsNil()

			By("FORWARD. check cc allowed balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance(item.Amount.String())
			}

			By("FORWARD. check industrial balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, "0")
			}

			By("BACKWARD")

			By("BACKWARD. channel transfer by customer backward")
			backwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			ts.TxInvokeWithSign(cmn.ChannelCC, cmn.ChannelCC, user1, "channelMultiTransferByCustomer", "",
				client.NewNonceByTime().Get(), id2, "INDUSTRIAL", string(backwardItemsJSON)).CheckErrorIsNil()

			By("BACKWARD. check cc allowed balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance("0")
			}

			By("BACKWARD. check industrial balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, "0")
			}

			By("BACKWARD. get channel transfer from")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferFrom", id2).CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("BACKWARD. create cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "createCCTransferTo", from).CheckErrorIsNil()

			By("BACKWARD. check industrial allowed balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, item.Amount.String())
			}
			By("BACKWARD. check cc allowed balance after createCCTransferTo")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance("0")
			}

			By("BACKWARD. channel transfer to")
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "channelTransferTo", id2).CheckResponseWithFunc(fChTrTo)

			By("BACKWARD. commit cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2).CheckErrorIsNil()

			By("BACKWARD. delete cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "removeCCTransferTo", id2).CheckErrorIsNil()

			By("BACKWARD. delete cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2).CheckErrorIsNil()

			By("BACKWARD. check cc allowed balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance("0")
			}

			By("BACKWARD. check industrial balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, item.Amount.String())
			}
		})

		It("multi transfer by admin success", func() {
			By("FORWARD")

			By("FORWARD. channel transfer forward")
			forwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			ts.TxInvokeWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, ts.Admin(), "channelMultiTransferByAdmin", "",
				client.NewNonceByTime().Get(), id, "CC", user1.AddressBase58Check, string(forwardItemsJSON)).CheckErrorIsNil()

			By("FORWARD. check industrial balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, "0")
			}

			By("FORWARD. check cc allowed balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance("0")
			}

			By("FORWARD. get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)

				return ""
			}
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "channelTransferFrom", id).CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("FORWARD. create cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "createCCTransferTo", from).CheckErrorIsNil()

			By("FORWARD. check industrial balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, "0")
			}

			By("FORWARD. check cc allowed after createCCTransferTo")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance(item.Amount.String())
			}

			By("FORWARD. channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferTo", id).CheckResponseWithFunc(fChTrTo)

			By("FORWARD. commit cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "commitCCTransferFrom", id).CheckErrorIsNil()

			By("FORWARD. delete cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "removeCCTransferTo", id).CheckErrorIsNil()

			By("FORWARD. delete cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "deleteCCTransferFrom", id).CheckErrorIsNil()

			By("FORWARD. check cc allowed balance. after deleteCCTransferFrom")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance(item.Amount.String())
			}

			By("FORWARD. check industrial allowed balance. after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, "0")
			}

			By("BACKWARD")

			By("BACKWARD. channel transfer by customer backward")
			backwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			ts.TxInvokeWithSign(cmn.ChannelCC, cmn.ChannelCC, ts.Admin(), "channelMultiTransferByAdmin", "",
				client.NewNonceByTime().Get(), id2, "INDUSTRIAL", user1.AddressBase58Check, string(backwardItemsJSON)).CheckErrorIsNil()

			By("BACKWARD. check cc allowed balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance("0")
			}

			By("BACKWARD. check industrial balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, "0")
			}

			By("BACKWARD. get channel transfer from")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferFrom", id2).CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("BACKWARD. create cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "createCCTransferTo", from).CheckErrorIsNil()

			By("BACKWARD. check cc balance after createCCTransferTo")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance("0")
			}

			By("BACKWARD. check industrial balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, item.Amount.String())
			}

			By("BACKWARD. channel transfer to")
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "channelTransferTo", id2).CheckResponseWithFunc(fChTrTo)

			By("BACKWARD. commit cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2).CheckErrorIsNil()

			By("BACKWARD. delete cc transfer to")
			ts.TxInvokeByRobot(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "removeCCTransferTo", id2).CheckErrorIsNil()

			By("BACKWARD. delete cc transfer from")
			ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2).CheckErrorIsNil()

			By("BACKWARD. check cc allowed balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).CheckBalance("0")
			}

			By("BACKWARD. check industrial balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).CheckIndustrialBalance(group, item.Amount.String())
			}
		})
	})
})
