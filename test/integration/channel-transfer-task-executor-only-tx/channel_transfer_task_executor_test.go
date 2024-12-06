package channel_transfer_task_executor_only_tx

import (
	"encoding/json"
	"strings"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Channel transfer with task executor only tx foundation Tests", func() {
	var (
		channels = []string{cmn.ChannelACL, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
		ts       client.TestSuite
	)

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})
	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	BeforeEach(func() {
		ts.InitNetwork(channels, integration.LedgerPort)
		ts.DeployChaincodes()
	})

	Describe("channel transfer test", func() {
		var (
			user1                *mocks.UserFoundation
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
			user1, err = mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			ts.AddUser(user1)

			id = uuid.NewString()
			id2 = uuid.NewString()

			By("emit tokens 1000")
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"emit", user1.AddressBase58Check, emitAmount)

			By("emit check")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(emitAmount)

			By("initialize industrial")
			ts.ExecuteTaskWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, ts.Admin(),
				"initialize")

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
				ts.ExecuteTaskWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, ts.Admin(),
					"transferIndustrial",
					user1.AddressBase58Check, group, item.Amount.String(), "comment")
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, item.Amount.String())
			}
		})

		It("by customer success", func() {
			By("FORWARD")

			By("channel transfer by customer forward")
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"channelTransferByCustomer",
				id, "CC", "FIAT", transferAmount)

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
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferFrom", id).
				CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "createCCTransferTo", from)

			By("check allowed balance 1")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance(transferAmount)

			By("channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferTo", id).
				CheckResponseWithFunc(fChTrTo)

			By("commit cc transfer from")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "commitCCTransferFrom", id)

			By("delete cc transfer to")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferTo", id)

			By("delete cc transfer from")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferFrom", id)

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance(transferAmount)

			By("check fiat balance")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(balanceAfterTransfer)

			By("BACKWARD")

			By("channel transfer by customer backward")
			ts.ExecuteTaskWithSign(cmn.ChannelCC, cmn.ChannelCC, user1,
				"channelTransferByCustomer",
				id2, "FIAT", "FIAT", transferAmount)

			By("check allowed balance after transfer")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance("0")

			By("get channel transfer from")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferFrom", id2).
				CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "createCCTransferTo", from)

			By("check fiat balance 1")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(emitAmount)

			By("channel transfer to")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferTo", id2).
				CheckResponseWithFunc(fChTrTo)

			By("commit cc transfer from")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2)

			By("delete cc transfer to")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferTo", id2)

			By("delete cc transfer from")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2)

			By("check allowed balance")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance("0")

			By("check fiat balance 2")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(emitAmount)
		})

		It("channel transfer by admin success", func() {
			By("FORWARD")

			By("channel transfer forward")
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"channelTransferByAdmin",
				id, "CC", user1.AddressBase58Check, "FIAT", transferAmount)

			By("check balance after transfer")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(balanceAfterTransfer)

			By("get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)

				return ""
			}
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferFrom", id).
				CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "createCCTransferTo", from)

			By("check allowed balance 1")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance(transferAmount)

			By("channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferTo", id).
				CheckResponseWithFunc(fChTrTo)

			By("commit cc transfer from")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "commitCCTransferFrom", id)

			By("delete cc transfer to")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferTo", id)

			By("delete cc transfer from")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferFrom", id)

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance(transferAmount)

			By("check fiat balance")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(balanceAfterTransfer)

			By("BACKWARD")

			By("channel transfer by admin backward")
			ts.ExecuteTaskWithSign(cmn.ChannelCC, cmn.ChannelCC, ts.Admin(),
				"channelTransferByAdmin",
				id2, "FIAT", user1.AddressBase58Check, "FIAT", transferAmount)

			By("check allowed balance after transfer")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance("0")

			By("get channel transfer from")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferFrom", id2).
				CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "createCCTransferTo", from)

			By("check fiat balance 1")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(emitAmount)

			By("channel transfer to")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferTo", id2).
				CheckResponseWithFunc(fChTrTo)

			By("commit cc transfer from")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2)

			By("delete cc transfer to")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferTo", id2)

			By("delete cc transfer from")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2)

			By("check allowed balance")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance("0")

			By("check fiat balance 2")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(emitAmount)
		})

		It("cancel forward success", func() {
			By("cancel channel transfer forward")
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"channelTransferByCustomer",
				id, "CC", "FIAT", transferAmount)

			By("check balance after transfer")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(balanceAfterTransfer)

			By("get channel transfer from")
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferFrom", id).
				CheckResponseWithFunc(fChTrFrom)

			By("cancel cc transfer from")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "cancelCCTransferFrom", id)

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance("0")

			By("check fiat balance")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(emitAmount)
		})

		It("cancel backward success", func() {
			By("FORWARD")

			By("channel transfer by customer forward")
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"channelTransferByCustomer",
				id, "CC", "FIAT", transferAmount)

			By("check balance after transfer")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(balanceAfterTransfer)

			By("get channel transfer from")
			from := ""
			fChTrFrom := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}
				from = string(out)
				return ""
			}
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransferFrom", id).
				CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("create cc transfer to")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "createCCTransferTo", from)

			By("check allowed balance 1")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance(transferAmount)

			By("channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferTo", id).
				CheckResponseWithFunc(fChTrTo)

			By("commit cc transfer from")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "commitCCTransferFrom", id)

			By("delete cc transfer to")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferTo", id)

			By("delete cc transfer from")
			ts.ExecuteTask(cmn.ChannelFiat, cmn.ChannelFiat, "deleteCCTransferFrom", id)

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance(transferAmount)

			By("check fiat balance")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(balanceAfterTransfer)

			By("BACKWARD")

			By("channel transfer by customer backward")
			ts.ExecuteTaskWithSign(cmn.ChannelCC, cmn.ChannelCC, user1,
				"channelTransferByCustomer",
				id2, "FIAT", "FIAT", transferAmount)

			By("check allowed balance after transfer")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance("0")

			By("get channel transfer from")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferFrom", id2).
				CheckResponseWithFunc(fChTrTo)

			By("cancel cc transfer from")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "cancelCCTransferFrom", id2)

			By("check allowed balance 2")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, "FIAT").
				CheckBalance(transferAmount)

			By("check fiat balance")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(balanceAfterTransfer)
		})

		It("query all transfers from", func() {
			transferAmount = "100"
			ids := make(map[string]struct{})

			By("channel transfer by customer forward1")
			id = uuid.NewString()
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"channelTransferByCustomer",
				id, "CC", "FIAT", transferAmount)
			ids[id] = struct{}{}

			By("channel transfer by customer forward2")
			id = uuid.NewString()
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"channelTransferByCustomer",
				id, "CC", "FIAT", transferAmount)
			ids[id] = struct{}{}

			By("channel transfer by customer forward3")
			id = uuid.NewString()
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"channelTransferByCustomer",
				id, "CC", "FIAT", transferAmount)
			ids[id] = struct{}{}

			By("channel transfer by customer forward4")
			id = uuid.NewString()
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"channelTransferByCustomer",
				id, "CC", "FIAT", transferAmount)
			ids[id] = struct{}{}

			By("channel transfer by customer forward5")
			id = uuid.NewString()
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, user1,
				"channelTransferByCustomer",
				id, "CC", "FIAT", transferAmount)
			ids[id] = struct{}{}

			By("check balance after transfer")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance("500")

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
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransfersFrom", "2", bookmark).
				CheckResponseWithFunc(fSize)

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
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransfersFrom", "1000", bookmark).
				CheckResponseWithFunc(fCheckIds)

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

				ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "channelTransfersFrom", "2", bookmark).
					CheckResponseWithFunc(fCheckBookmark)

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
			ts.ExecuteTaskWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, user1,
				"channelMultiTransferByCustomer",
				id, "CC", string(forwardItemsJSON))

			By("FORWARD. check industrial balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, "0")
			}

			By("FORWARD. check cc allowed balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance("0")
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
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "channelTransferFrom", id).
				CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("FORWARD. create cc transfer to")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "createCCTransferTo", from)

			By("FORWARD. check industrial balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, "0")
			}

			By("FORWARD. check cc allowed balance after createCCTransferTo")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance(item.Amount.String())
			}

			By("FORWARD. channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferTo", id).
				CheckResponseWithFunc(fChTrTo)

			By("FORWARD. commit cc transfer from")
			ts.ExecuteTask(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "commitCCTransferFrom", id)

			By("FORWARD. delete cc transfer to")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferTo", id)

			By("FORWARD. delete cc transfer from")
			ts.ExecuteTask(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "deleteCCTransferFrom", id)

			By("FORWARD. check cc allowed balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance(item.Amount.String())
			}

			By("FORWARD. check industrial balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, "0")
			}

			By("BACKWARD")

			By("BACKWARD. channel transfer by customer backward")
			backwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			ts.ExecuteTaskWithSign(cmn.ChannelCC, cmn.ChannelCC, user1,
				"channelMultiTransferByCustomer",
				id2, "INDUSTRIAL", string(backwardItemsJSON))

			By("BACKWARD. check cc allowed balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance("0")
			}

			By("BACKWARD. check industrial balance after channelMultiTransferByCustomer")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, "0")
			}

			By("BACKWARD. get channel transfer from")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferFrom", id2).
				CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("BACKWARD. create cc transfer to")
			ts.ExecuteTask(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "createCCTransferTo", from)

			By("BACKWARD. check industrial allowed balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, item.Amount.String())
			}
			By("BACKWARD. check cc allowed balance after createCCTransferTo")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance("0")
			}

			By("BACKWARD. channel transfer to")
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "channelTransferTo", id2).
				CheckResponseWithFunc(fChTrTo)

			By("BACKWARD. commit cc transfer from")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2)

			By("BACKWARD. delete cc transfer to")
			ts.ExecuteTask(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "deleteCCTransferTo", id2)

			By("BACKWARD. delete cc transfer from")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2)

			By("BACKWARD. check cc allowed balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance("0")
			}

			By("BACKWARD. check industrial balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, item.Amount.String())
			}
		})

		It("multi transfer by admin success", func() {
			By("FORWARD")

			By("FORWARD. channel transfer forward")
			forwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			ts.ExecuteTaskWithSign(cmn.ChannelIndustrial, cmn.ChannelIndustrial, ts.Admin(),
				"channelMultiTransferByAdmin",
				id, "CC", user1.AddressBase58Check, string(forwardItemsJSON))

			By("FORWARD. check industrial balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, "0")
			}

			By("FORWARD. check cc allowed balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance("0")
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
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "channelTransferFrom", id).
				CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("FORWARD. create cc transfer to")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "createCCTransferTo", from)

			By("FORWARD. check industrial balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, "0")
			}

			By("FORWARD. check cc allowed after createCCTransferTo")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance(item.Amount.String())
			}

			By("FORWARD. channel transfer to")
			fChTrTo := func(out []byte) string {
				if len(out) == 0 {
					return "out is empty"
				}

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferTo", id).
				CheckResponseWithFunc(fChTrTo)

			By("FORWARD. commit cc transfer from")
			ts.ExecuteTask(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "commitCCTransferFrom", id)

			By("FORWARD. delete cc transfer to")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferTo", id)

			By("FORWARD. delete cc transfer from")
			ts.ExecuteTask(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "deleteCCTransferFrom", id)

			By("FORWARD. check cc allowed balance. after deleteCCTransferFrom")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance(item.Amount.String())
			}

			By("FORWARD. check industrial allowed balance. after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, "0")
			}

			By("BACKWARD")

			By("BACKWARD. channel transfer by customer backward")
			backwardItemsJSON, err := json.Marshal(transferItems)
			Expect(err).NotTo(HaveOccurred())
			ts.ExecuteTaskWithSign(cmn.ChannelCC, cmn.ChannelCC, ts.Admin(),
				"channelMultiTransferByAdmin",
				id2, "INDUSTRIAL", user1.AddressBase58Check, string(backwardItemsJSON))

			By("BACKWARD. check cc allowed balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance("0")
			}

			By("BACKWARD. check industrial balance after channelMultiTransferByAdmin")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, "0")
			}

			By("BACKWARD. get channel transfer from")
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "channelTransferFrom", id2).
				CheckResponseWithFunc(fChTrFrom)
			Expect(from).NotTo(BeEmpty())

			By("BACKWARD. create cc transfer to")
			ts.ExecuteTask(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "createCCTransferTo", from)

			By("BACKWARD. check cc balance after createCCTransferTo")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance("0")
			}

			By("BACKWARD. check industrial balance after createCCTransferTo")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, item.Amount.String())
			}

			By("BACKWARD. channel transfer to")
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "channelTransferTo", id2).
				CheckResponseWithFunc(fChTrTo)

			By("BACKWARD. commit cc transfer from")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "commitCCTransferFrom", id2)

			By("BACKWARD. delete cc transfer to")
			ts.ExecuteTask(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "deleteCCTransferTo", id2)

			By("BACKWARD. delete cc transfer from")
			ts.ExecuteTask(cmn.ChannelCC, cmn.ChannelCC, "deleteCCTransferFrom", id2)

			By("BACKWARD. check cc allowed balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				ts.Query(cmn.ChannelCC, cmn.ChannelCC, "allowedBalanceOf", user1.AddressBase58Check, item.Token).
					CheckBalance("0")
			}

			By("BACKWARD. check industrial balance after deleteCCTransferFrom")
			for _, item := range transferItems {
				group := strings.Split(item.Token, "_")[1]
				ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "industrialBalanceOf", user1.AddressBase58Check).
					CheckIndustrialBalance(group, item.Amount.String())
			}
		})
	})
})
