package basic

import (
	"encoding/json"
	"github.com/hyperledger/fabric/integration"
	"os"
	"syscall"
	"time"

	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	docker "github.com/fsouza/go-dockerclient"
	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Basic foundation tests with different key types", func() {
	var (
		testDir          string
		cli              *docker.Client
		networkProcess   ifrit.Process
		ordererProcesses []ifrit.Process
		peerProcesses    ifrit.Process
		ts               client.TestSuite
	)

	BeforeEach(func() {
		networkProcess = nil
		ordererProcesses = nil
		peerProcesses = nil
		var err error
		testDir, err = os.MkdirTemp("", "foundation")
		Expect(err).NotTo(HaveOccurred())

		cli, err = docker.NewClientFromEnv()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if networkProcess != nil {
			networkProcess.Signal(syscall.SIGTERM)
			Eventually(networkProcess.Wait(), ts.Network().EventuallyTimeout).Should(Receive())
		}
		if peerProcesses != nil {
			peerProcesses.Signal(syscall.SIGTERM)
			Eventually(peerProcesses.Wait(), ts.Network().EventuallyTimeout).Should(Receive())
		}
		if ts.Network() != nil {
			ts.Network().Cleanup()
		}
		for _, ordererInstance := range ordererProcesses {
			ordererInstance.Signal(syscall.SIGTERM)
			Eventually(ordererInstance.Wait(), ts.Network().EventuallyTimeout).Should(Receive())
		}
		err := os.RemoveAll(testDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("foundation test", func() {
		var (
			channels     = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
			redisProcess ifrit.Process
			redisDB      *runner.RedisDB
			robotProc    ifrit.Process
		)
		BeforeEach(func() {
			By("start redis")
			redisDB = &runner.RedisDB{}
			redisProcess = ifrit.Invoke(redisDB)
			Eventually(redisProcess.Ready(), runnerFbk.DefaultStartTimeout).Should(BeClosed())
			Consistently(redisProcess.Wait()).ShouldNot(Receive())
		})
		BeforeEach(func() {
			ts = client.NewTestSuite("", "", "", "", redisDB.Address(), channels, testDir, cli, integration.DevModePort, components, ordererProcesses, peerProcesses)
			ts.DeployChannels()
		})
		BeforeEach(func() {
			By("start robot")
			robotRunner := ts.NetworkFound().RobotRunner()
			robotProc = ifrit.Invoke(robotRunner)
			Eventually(robotProc.Ready(), ts.Network().EventuallyTimeout).Should(BeClosed())
		})
		AfterEach(func() {
			By("stop robot")
			if robotProc != nil {
				robotProc.Signal(syscall.SIGTERM)
				Eventually(robotProc.Wait(), ts.Network().EventuallyTimeout).Should(Receive())
			}
		})
		AfterEach(func() {
			By("stop redis " + redisDB.Address())
			if redisProcess != nil {
				redisProcess.Signal(syscall.SIGTERM)
				Eventually(redisProcess.Wait(), time.Minute).Should(Receive())
			}
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
			ts.AddUser(ts.Admin())

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
