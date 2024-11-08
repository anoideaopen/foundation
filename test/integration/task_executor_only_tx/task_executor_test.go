package task_executor_only_tx

import (
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Task Executor only tx foundation Tests", func() {
	var (
		channels = []string{cmn.ChannelAcl, cmn.ChannelFiat}
		ts       client.TestSuite
	)

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})
	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	BeforeEach(func() {
		ts.InitNetwork(channels, integration.IdemixBasePort)
		ts.DeployChaincodes()
	})

	Describe("task executor test", func() {
		var (
			user1 *mocks.UserFoundation
		)

		BeforeEach(func() {
			By("add admin to acl")
			ts.AddAdminToACL()

			By("add user to acl")
			var err error
			user1, err = mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())
			ts.AddUser(user1)
		})

		It("execute tasks with tx", func() {
			var (
				emitAmount = "1000"
			)

			By("emit tokens 1000")
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"emit", user1.AddressBase58Check, emitAmount)

			By("emit check")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).
				CheckBalance(emitAmount)
		})

		It("execute batch with nbtx", func() {
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(), "healthCheckNb")
		})

		It("execute tasks with tx", func() {
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(), "healthCheck")
		})

		It("execute tasks with tx and nbtx", func() {
			tasks := make([]*pbfound.Task, 0)
			task, err := client.CreateTaskWithSignArgs("healthCheckNb", cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin())
			Expect(err).NotTo(HaveOccurred())
			tasks = append(tasks, task)
			task, err = client.CreateTaskWithSignArgs("healthCheck", cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin())
			Expect(err).NotTo(HaveOccurred())
			tasks = append(tasks, task)

			ts.ExecuteTasks(cmn.ChannelFiat, cmn.ChannelFiat, tasks...)
		})
	})
})
