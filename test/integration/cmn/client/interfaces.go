package client

import (
	"github.com/anoideaopen/acl/cc"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn/client/types"
	"github.com/anoideaopen/robot/helpers/ntesting"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/integration"
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

type ACLAddUserInterface interface {
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
}

type ACLListsInterface interface {
	// AddToBlackList adds user to blacklist
	AddToBlackList(user *UserFoundation)
	// AddToGrayList adds user to graylist
	AddToGrayList(user *UserFoundation)
	// DelFromBlackList adds user to a black list
	DelFromBlackList(user *UserFoundation)
	// DelFromGrayList adds user to a gray list
	DelFromGrayList(user *UserFoundation)
	// CheckUserInList - checks if user in gray or black list
	CheckUserInList(listType cc.ListType, user *UserFoundation)
	// CheckUserNotInList - checks if user in gray or black list
	CheckUserNotInList(listType cc.ListType, user *UserFoundation)
}

type ACLKeysInterface interface {
	// ChangePublicKey - changes user public key by validators
	ChangePublicKey(user *UserFoundation, newPubKeyBase58 string, reason string, reasonID string, validators ...*UserFoundation)
	// ChangePublicKeyBase58signed - changes user public key by validators with base58 signatures
	ChangePublicKeyBase58signed(user *UserFoundation, requestID string, chaincodeName string, channelID string, newPubKeyBase58 string, reason string, reasonID string, validators ...*UserFoundation)
	// ChangeMultisigPublicKey changes public key for multisigned user by validators
	ChangeMultisigPublicKey(multisignedUser *UserFoundationMultisigned, oldPubKeyBase58 string, newPubKeyBase58 string, reason string, reasonID string, validators ...*UserFoundation)
	// CheckUserChangedKey checks if user changed key
	CheckUserChangedKey(newPublicKeyBase58Check, oldAddressBase58Check string)
}

type ACLInterface interface {
	ACLAddUserInterface
	ACLListsInterface
	ACLKeysInterface
	AddRights(channelName, chaincodeName, role, operation string, user *UserFoundation)
	// RemoveRights removes right for defined user with specified role and operation to ACL channel
	RemoveRights(channelName, chaincodeName, role, operation string, user *UserFoundation)
	// CheckAccountInfo checks account info
	CheckAccountInfo(user *UserFoundation, kycHash string, isGrayListed, isBlackListed bool)
	// SetAccountInfo sets account info
	SetAccountInfo(user *UserFoundation, kycHash string, isGrayListed, isBlackListed bool)
	// SetKYC sets kyc hash
	SetKYC(user *UserFoundation, kycHash string, validators ...*UserFoundation)
}

type StarterInterface interface {
	// StartRedis starts testsuite redis
	StartRedis()
	// StartRobot starts testsuite robot
	StartRobot()
	// StartChannelTransfer starts testsuite channel transfer
	StartChannelTransfer()
	// InitNetwork initializes testsuite network
	InitNetwork(channels []string, testPort integration.TestPortRange, opts ...NetworkOption)
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
	// TestDir returns testsuite temporary test directory
	TestDir() string
	// DockerClient returns testsuite docker client
	DockerClient() *docker.Client
	// CiData return CiData for robot testing
	CiData(opts ...CiDataOption) ntesting.CiTestData
}

type TaskExecutor interface {
	ExecuteTask(channel string, chaincode string, method string, args ...string) string
	ExecuteTasks(channel string, chaincode string, tasks ...*pbfound.Task) string
	ExecuteTaskWithSign(channel string, chaincode string, user *UserFoundation, method string, args ...string) string
}

type TestSuite interface {
	InvokeInterface
	QueryInterface
	ACLInterface
	StarterInterface
	StopperInterface
	FieldGetter
	TaskExecutor
}
