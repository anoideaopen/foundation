package client

import (
	"fmt"
	"time"

	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client/types"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func (ts *FoundationTestSuite) Query(channelName, chaincodeName string, args ...string) *types.QueryResult {
	result := &types.QueryResult{}
	Eventually(func() *types.QueryResult {
		sess, err := ts.Network.PeerUserSession(
			ts.Peer,
			ts.MainUserName,
			commands.ChaincodeQuery{
				ChannelID: channelName,
				Name:      chaincodeName,
				Ctor:      cmn.CtorFromSlice(args),
			},
		)
		Eventually(sess, ts.Network.EventuallyTimeout).Should(gexec.Exit())
		Expect(err).NotTo(HaveOccurred())

		result.SetErrorCode(int32(sess.ExitCode()))
		result.SetResponse(sess.Out.Contents())
		result.SetMessage(sess.Err.Contents())

		return result
	}, ts.Network.EventuallyTimeout, time.Second).Should(Not(BeNil()))

	return result
}

func (ts *FoundationTestSuite) QueryWithSign(
	channelName string,
	chaincodeName string,
	user *mocks.UserFoundation,
	fn string,
	requestID string,
	nonce string,
	args ...string,
) *types.QueryResult {
	ctorArgs := append(append([]string{fn, requestID, channelName, chaincodeName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	return ts.Query(channelName, chaincodeName, ctorArgs...)
}

type SwapFunctionName string

const (
	SfnSwapGet      SwapFunctionName = "swapGet"
	SfnMultiSwapGet SwapFunctionName = "multiSwapGet"
)

func (ts *FoundationTestSuite) SwapGet(channelName, chaincodeName string, functionName SwapFunctionName, swapBeginTxID string) *types.QueryResult {
	result := &types.QueryResult{}
	Eventually(func() string {
		sess, err := ts.Network.PeerUserSession(
			ts.Peer,
			ts.MainUserName,
			commands.ChaincodeQuery{
				ChannelID: channelName,
				Name:      chaincodeName,
				Ctor:      cmn.CtorFromSlice([]string{string(functionName), swapBeginTxID}),
			},
		)
		Eventually(sess, ts.Network.EventuallyTimeout).Should(gexec.Exit())
		Expect(err).NotTo(HaveOccurred())

		if sess.ExitCode() != 0 && sess.Err.Contents() != nil {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		if len(sess.Out.Contents()) == 0 {
			return "out is empty"
		}

		result.SetErrorCode(int32(sess.ExitCode()))
		result.SetResponse(sess.Out.Contents())
		result.SetMessage(sess.Err.Contents())

		return ""
	}, ts.Network.EventuallyTimeout, time.Second).Should(BeEmpty())

	return result
}

func (ts *FoundationTestSuite) Metadata(channelName, chaincodeName string) *types.QueryResult {
	result := &types.QueryResult{}
	sess, err := ts.Network.PeerUserSession(
		ts.Peer,
		ts.MainUserName,
		commands.ChaincodeQuery{
			ChannelID: channelName,
			Name:      chaincodeName,
			Ctor:      cmn.CtorFromSlice([]string{"metadata"}),
		})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.Network.EventuallyTimeout).Should(gexec.Exit(0))

	result.SetErrorCode(int32(sess.ExitCode()))
	result.SetResponse(sess.Out.Contents())
	result.SetMessage(sess.Err.Contents())

	return result
}
