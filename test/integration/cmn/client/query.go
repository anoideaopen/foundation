package client

import (
	"fmt"
	"time"

	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client/types"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type CheckResultFunc func(err error, exitCode int, sessError []byte, sessOut []byte) string

//	func exampleCheck() CheckResultFunc {
//		return func(err error, sessExitCode int, sessError []byte, sessOut []byte) string {
//			if err != nil {
//				return fmt.Sprintf("error executing command: %v", err)
//			}
//
//			if sessExitCode != 0 {
//				return fmt.Sprintf("exit code is %d: %s, %v", sessExitCode, string(sessError), err)
//			}
//
//			out := sessOut[:len(sessOut)-1] // skip line feed
//			resp := &pb.Response{}
//			err = proto.Unmarshal(out, resp)
//			if err != nil {
//				return fmt.Sprintf("failed to unmarshal response: %v", err)
//			}
//
//			// check response
//			if ... != ... {
//				return fmt.Sprintf("error: expected %s, received %s", ..., ...)
//			}
//
//			return ""
//		}
//	}

// Deprecated: need to remove after migrating to testsuite
// Query func for query from foundation fabric
func Query(network *nwo.Network, peer *nwo.Peer, channel string, ccName string,
	checkResultFunc CheckResultFunc, args ...string) {
	Eventually(func() string {
		sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
			ChannelID: channel,
			Name:      ccName,
			Ctor:      cmn.CtorFromSlice(args),
		})
		Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit())

		return checkResultFunc(err, sess.ExitCode(), sess.Err.Contents(), sess.Out.Contents())
	}, network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

// Deprecated: need to remove after migrating to testsuite
// QueryWithSign func for query with sign from foundation fabric
func QueryWithSign(
	network *nwo.Network,
	peer *nwo.Peer,
	channel string,
	ccName string,
	checkResultFunc CheckResultFunc,
	user *UserFoundation,
	fn string,
	requestID string,
	nonce string,
	args ...string,
) {
	ctorArgs := append(append([]string{fn, requestID, channel, ccName}, args...), nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	ctorArgs = append(ctorArgs, pubKey, base58.Encode(sMsg))
	Query(network, peer, channel, ccName, checkResultFunc, ctorArgs...)
}

func (ts *testSuite) Query(channelName, chaincodeName string, args ...string) *types.QueryResult {
	result := &types.QueryResult{}
	Eventually(func() *types.QueryResult {
		sess, err := ts.network.PeerUserSession(
			ts.peer,
			ts.mainUserName,
			commands.ChaincodeQuery{
				ChannelID: channelName,
				Name:      chaincodeName,
				Ctor:      cmn.CtorFromSlice(args),
			},
		)
		Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit())
		Expect(err).NotTo(HaveOccurred())

		result.SetErrorCode(int32(sess.ExitCode()))
		result.SetResponse(sess.Out.Contents())
		result.SetMessage(sess.Err.Contents())

		return result
	}, ts.network.EventuallyTimeout, time.Second).Should(Not(BeNil()))

	return result
}

func (ts *testSuite) QueryWithSign(
	channelName string,
	chaincodeName string,
	user *UserFoundation,
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

func (ts *testSuite) SwapGet(channelName, chaincodeName string, functionName SwapFunctionName, swapBeginTxID string) *types.QueryResult {
	result := &types.QueryResult{}
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(
			ts.peer,
			ts.mainUserName,
			commands.ChaincodeQuery{
				ChannelID: channelName,
				Name:      chaincodeName,
				Ctor:      cmn.CtorFromSlice([]string{string(functionName), swapBeginTxID}),
			},
		)
		Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit())
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
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())

	return result
}

func (ts *testSuite) Metadata(channelName, chaincodeName string) *types.QueryResult {
	result := &types.QueryResult{}
	sess, err := ts.network.PeerUserSession(
		ts.peer,
		ts.mainUserName,
		commands.ChaincodeQuery{
			ChannelID: channelName,
			Name:      chaincodeName,
			Ctor:      cmn.CtorFromSlice([]string{"metadata"}),
		})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))

	result.SetErrorCode(int32(sess.ExitCode()))
	result.SetResponse(sess.Out.Contents())
	result.SetMessage(sess.Err.Contents())

	return result
}
