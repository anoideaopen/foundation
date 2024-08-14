package client

import (
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/tedsuo/ifrit"
)

/*
type TxInterface interface {
	Invoke(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string) types.InvokeResult
	WithSigh(request *RequestInterface)
	ByRobot(request *RequestInterface)
}

type RequestInterface interface {
	Invoke(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string) types.InvokeResult
	Query() types.QueryResult
}
*/

type TestSuite interface {
	// Admin returns testsuite admin
	Admin() *UserFoundation
	// FeeSetter returns testsuite fee setter user
	FeeSetter() *UserFoundation
	// FeeAddressSetter returns testsuite fee address setter
	FeeAddressSetter() *UserFoundation
	// Network returns testsuite network
	Network() *nwo.Network
	// NetworkFound returns testsuite network foundation
	NetworkFound() *cmn.NetworkFoundation
	// Peer returns testsuite peer
	Peer() *nwo.Peer
	// OrdererProcesses returns testsuite orderer processes
	OrdererProcesses() []ifrit.Process
	// PeerProcess returns testsuite peer process
	PeerProcess() ifrit.Process
	// StartRedis starts testsuite redis
	StartRedis()
	// StopRedis stops testsuite redis
	StopRedis()
	// StartRobot starts testsuite robot
	StartRobot()
	// StopRobot stops testsuite robot
	StopRobot()
	// StartChannelTransfer starts testsuite channel transfer
	StartChannelTransfer()
	// StopChannelTransfer stops testsuite channel transfer
	StopChannelTransfer()
	// InitNetwork initializes testsuite network
	InitNetwork(testPort integration.TestPortRange)
	// DeployChannels deploys channels to testsuite network
	DeployChannels()
	// ShutdownNetwork shuts down testsuite network
	ShutdownNetwork()
	// TxInvoke func for invoke to foundation fabric
	TxInvoke(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string) string
	// TxInvokeByRobot func for invoke to foundation fabric from robot
	TxInvokeByRobot(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string) string
	// TxInvokeWithSign func for invoke with sign to foundation fabric
	TxInvokeWithSign(channelName, chaincodeName string, user *UserFoundation, fn, requestID, nonce string, checkErr CheckResultFunc, args ...string) (txId string)
	// TxInvokeWithMultisign invokes transaction to foundation fabric with multisigned user
	TxInvokeWithMultisign(channelName, chaincodeName string, user *UserFoundationMultisigned, fn, requestID, nonce string, checkErr CheckResultFunc, args ...string) (txId string)
	// NBTxInvoke func for invoke to foundation fabric
	NBTxInvoke(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string)
	// NBTxInvokeByRobot func for invoke to foundation fabric from robot
	NBTxInvokeByRobot(channelName, chaincodeName string, checkErr CheckResultFunc, args ...string)
	// NBTxInvokeWithSign func for invoke with sign to foundation fabric
	NBTxInvokeWithSign(channelName, chaincodeName string, checkErr CheckResultFunc, user *UserFoundation, fn, requestID, nonce string, args ...string)
	// Query func for query from foundation fabric
	Query(channelName, chaincodeName string, checkResultFunc CheckResultFunc, args ...string)
	// QueryWithSign func for query with sign from foundation fabric
	QueryWithSign(channelName, chaincodeName string, checkResultFunc CheckResultFunc, user *UserFoundation, fn, requestID, nonce string, args ...string)
	// AddUser adds new user to ACL channel
	AddUser(user *UserFoundation)
	// AddAdminToACL adds testsuite admin to ACL channel
	AddAdminToACL()
	// AddFeeSetterToACL adds testsuite fee setter to ACL channel
	AddFeeSetterToACL()
	// AddFeeAddressSetterToACL adds testsuite fee address setter to ACL channel
	AddFeeAddressSetterToACL()
	// AddUserMultisigned adds multisigned user to ACL channel
	AddUserMultisigned(user *UserFoundationMultisigned)
	// AddRights adds right for defined user with specified role and operation to ACL channel
	AddRights(channelName, chaincodeName, role, operation string, user *UserFoundation)
	// RemoveRights removes right for defined user with specified role and operation to ACL channel
	RemoveRights(channelName, chaincodeName, role, operation string, user *UserFoundation)
	// ChangeMultisigPublicKey changes public key for multisigned user by validators
	ChangeMultisigPublicKey(multisignedUser *UserFoundationMultisigned, oldPubKeyBase58 string, newPubKeyBase58 string, reason string, reasonID string, validators ...*UserFoundation)
}
