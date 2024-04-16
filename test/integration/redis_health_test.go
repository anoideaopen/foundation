/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package integration

import (
	"os"
	"syscall"
	"time"

	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Health", func() {
	var (
		testDir string
	)

	BeforeEach(func() {
		var err error
		testDir, err = os.MkdirTemp("", "foundation")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(testDir)
	})

	Describe("Redis health checks", func() {
		It("returns appropriate response codes", func() {
			By("returning 200 when able to reach Redis DB")
			redisDB := &runner.RedisDB{}
			redisProcess := ifrit.Invoke(redisDB)
			Eventually(redisProcess.Ready(), runnerFbk.DefaultStartTimeout).Should(BeClosed())
			Consistently(redisProcess.Wait()).ShouldNot(Receive())
			redisAddr := redisDB.Address()

			By("stop redis " + redisAddr)
			redisProcess.Signal(syscall.SIGTERM)
			Eventually(redisProcess.Wait(), time.Minute).Should(Receive())
		})
	})
})
