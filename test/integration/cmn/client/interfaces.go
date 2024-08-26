package client

import (
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client/types"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo"
)

type InvokeInterface interface {
	// TxInvoke func for invoke to foundation fabric
	TxInvoke(channelName, chaincodeName string, args ...string) *types.InvokeResult
	// TxInvokeByRobot func for invoke to foundation fabric from robot
	TxInvokeByRobot(channelName, chaincodeName string, args ...string) *types.InvokeResult
	// TxInvokeWithSign func for invoke with sign to foundation fabric
	TxInvokeWithSign(channelName, chaincodeName string, user *UserFoundation, fn, requestID, nonce string, args ...string) *types.InvokeResult
	// TxInvokeWithMultisign invokes transaction to foundation fabric with multisigned user
	TxInvokeWithMultisign(channelName, chaincodeName string, user *UserFoundationMultisigned, fn, requestID, nonce string, args ...string) *types.InvokeResult
	// NBTxInvoke func for invoke to foundation fabric
	NBTxInvoke(channelName, chaincodeName string, args ...string) *types.InvokeResult
	// NBTxInvokeByRobot func for invoke to foundation fabric from robot
	NBTxInvokeByRobot(channelName, chaincodeName string, args ...string) *types.InvokeResult
	// NBTxInvokeWithSign func for invoke with sign to foundation fabric
	NBTxInvokeWithSign(channelName, chaincodeName string, user *UserFoundation, fn, requestID, nonce string, args ...string) *types.InvokeResult
}

type QueryInterface interface {
	// Query func for query from foundation fabric
	Query(channelName, chaincodeName string, args ...string) *types.QueryResult
	// QueryWithSign func for query with sign from foundation fabric
	QueryWithSign(channelName, chaincodeName string, user *UserFoundation, fn, requestID, nonce string, args ...string) *types.QueryResult
	// SwapGet requests specified channel for swap information until it appears
	SwapGet(channelName, chaincodeName string, functionName SwapFunctionName, swapBeginTxID string) *types.QueryResult
	// Metadata returns chaincode metadata
	Metadata(channelName, chaincodeName string) *types.QueryResult
}

type ACLInterface interface {
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

type StarterInterface interface {
	// StartRedis starts testsuite redis
	StartRedis()
	// StartRobot starts testsuite robot
	StartRobot()
	// StartChannelTransfer starts testsuite channel transfer
	StartChannelTransfer()
	// InitNetwork initializes testsuite network
	InitNetwork(channels []string, testPort integration.TestPortRange)
	// DeployChaincodes deploys chaincodes to testsuite network defined channels
	DeployChaincodes()
	// DeployChaincodesByName deploys chaincodes to testsuite channels
	DeployChaincodesByName(channels []string)
	// DeployFiat deploys FIAT chaincode to testsuite FIAT channel with specified addresses
	DeployFiat(adminAddress, feeSetterAddress, feeAddressSetterAddress string)
}

type StopperInterface interface {
	// StopRedis stops testsuite redis
	StopRedis()
	// StopRobot stops testsuite robot
	StopRobot()
	// StopChannelTransfer stops testsuite channel transfer
	StopChannelTransfer()
	// StopOrderers stop all orderer processes
	StopOrderers()
	// ShutdownNetwork shuts down testsuite network
	ShutdownNetwork()
}

type FieldGetter interface {
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
	// TestDir returns testsuite temporary test directory
	TestDir() string
	// DockerClient returns testsuite docker client
	DockerClient() *docker.Client
}

type TestSuite interface {
	InvokeInterface
	QueryInterface
	ACLInterface
	StarterInterface
	StopperInterface
	FieldGetter
}
