package version_and_nonce

import (
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Version and Nonce Tests", func() {
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
		By("start redis")
		ts.StartRedis()
	})

	BeforeEach(func() {
		ts.InitNetwork(channels, integration.DiscoveryBasePort)
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

	Describe("version tests", func() {
		It("build version", func() {
			fCheckBuildVersion := func(out []byte) string {
				resp := &debug.BuildInfo{}
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Path).To(Equal(cmn.CcModulePath()))

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "buildInfo").CheckResponseWithFunc(fCheckBuildVersion)
		})

		It("core chaincode id name", func() {
			fCheckChaincodeIDName := func(out []byte) string {
				var resp string
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeEmpty())

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "coreChaincodeIDName").CheckResponseWithFunc(fCheckChaincodeIDName)
		})

		It("system env", func() {
			fCheckSystemEnv := func(out []byte) string {
				resp := make(map[string]string)
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				_, ok := resp["/etc/issue"]
				Expect(ok).To(BeTrue())

				return ""
			}
			ts.Query(cmn.ChannelCC, cmn.ChannelCC, "systemEnv").CheckResponseWithFunc(fCheckSystemEnv)
		})

		It("embed src files", func() {
			By("get names of files chaincode")
			fCheckChaincodeNames := func(out []byte) string {
				var resp []string
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeEmpty())

				return ""
			}
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "nameOfFiles").CheckResponseWithFunc(fCheckChaincodeNames)

			By("get file of chaincode")
			fChaincodeSrcFile := func(out []byte) string {
				var resp string
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeEmpty())
				Expect(resp[8:23]).To(Equal("industrialtoken"))

				return ""
			}
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "srcFile", "industrial_token/token.go").CheckResponseWithFunc(fChaincodeSrcFile)

			By("get part file of chaincode")
			fChaincodeSrcPartFile := func(out []byte) string {
				var resp string
				err := json.Unmarshal(out, &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeEmpty())
				Expect(resp).To(Equal("industrialtoken"))

				return ""
			}
			ts.Query(cmn.ChannelIndustrial, cmn.ChannelIndustrial, "srcPartFile", "industrial_token/token.go", "8", "23").CheckResponseWithFunc(fChaincodeSrcPartFile)
		})
	})

	It("nonce test", func() {
		By("add admin to acl")
		ts.AddAdminToACL()

		By("add user to acl")
		user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user1)

		By("prepare nonces")
		nonce := client.NewNonceByTime()
		nonce1 := nonce.Get()
		nonce.Add(51000)
		nonce2 := nonce.Get()
		nonce.Next()
		nonce3 := nonce.Get()
		nonce.Next()
		nonce4 := nonce.Get()

		emitAmount := "1"

		By("emit tokens 1")
		ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
			"emit", "", nonce3, user1.AddressBase58Check, emitAmount).CheckErrorIsNil()

		By("emit tokens 2")
		ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
			"emit", "", nonce2, user1.AddressBase58Check, emitAmount).CheckErrorIsNil()

		By("NEGATIVE: emit tokens 3")
		ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
			"emit", "", nonce1, user1.AddressBase58Check, emitAmount).CheckErrorEquals(fmt.Sprintf("function and args loading error: incorrect nonce %s000000, less than %s000000", nonce1, nonce3))

		By("NEGATIVE: emit tokens 4")
		ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
			"emit", "", nonce3, user1.AddressBase58Check, emitAmount).CheckErrorEquals(fmt.Sprintf("function and args loading error: nonce %s000000 already exists", nonce3))

		By("emit tokens 5")
		ts.TxInvokeWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
			"emit", "", nonce4, user1.AddressBase58Check, emitAmount).CheckErrorIsNil()

		By("emit check")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, "balanceOf", user1.AddressBase58Check).CheckBalance("3")
	})
})
