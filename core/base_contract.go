package core

import (
	"embed"
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"sort"
	"strconv"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/version"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// BaseContract is a base contract for all contracts
type BaseContract struct {
	id          string
	stub        shim.ChaincodeStubInterface
	methods     []string
	atomyzeSKI  []byte
	initArgs    []string
	noncePrefix StateKey
	srcFs       *embed.FS
}

func (bc *BaseContract) baseContractInit(cc BaseContractInterface) { //nolint:unused
	bc.id = cc.GetID()
}

func (bc *BaseContract) setSrcFs(srcFs *embed.FS) { //nolint:unused
	bc.srcFs = srcFs
}

// GetStub returns stub
func (bc *BaseContract) GetStub() shim.ChaincodeStubInterface {
	return bc.stub
}

// GetMethods returns list of methods
func (bc *BaseContract) GetMethods() []string {
	return bc.methods
}

func (bc *BaseContract) addMethod(mm string) { //nolint:unused
	bc.methods = append(bc.methods, mm)
	sort.Strings(bc.methods)
}

func (bc *BaseContract) setStubAndInitArgs( //nolint:unused
	stub shim.ChaincodeStubInterface,
	atomyzeSKI []byte,
	args []string,
	noncePrefix StateKey,
) {
	bc.stub = stub
	bc.atomyzeSKI = atomyzeSKI
	bc.initArgs = args
	bc.noncePrefix = noncePrefix
}

// GetAtomyzeSKI returns atomyzeSKI
func (bc *BaseContract) GetAtomyzeSKI() []byte {
	return bc.atomyzeSKI
}

// GetInitArg returns init arg by index
func (bc *BaseContract) GetInitArg(idx int) string {
	if len(bc.initArgs) > 0 && idx < len(bc.initArgs) {
		return bc.initArgs[idx]
	}

	return ""
}

// GetInitArgsLen returns init args len
func (bc *BaseContract) GetInitArgsLen() int {
	return len(bc.initArgs)
}

// QueryGetNonce returns nonce
func (bc *BaseContract) QueryGetNonce(owner *types.Address) (string, error) {
	prefix := hex.EncodeToString([]byte{byte(bc.noncePrefix)})
	key, err := bc.stub.CreateCompositeKey(prefix, []string{owner.String()})
	if err != nil {
		return "", err
	}

	data, err := bc.stub.GetState(key)
	if err != nil {
		return "", err
	}

	exist := new(big.Int).String()

	lastNonce := new(pb.Nonce)
	if len(data) > 0 {
		if err = proto.Unmarshal(data, lastNonce); err != nil {
			// let's just say it's an old nonsense
			lastNonce.Nonce = []uint64{new(big.Int).SetBytes(data).Uint64()}
		}
		exist = strconv.FormatUint(lastNonce.Nonce[len(lastNonce.Nonce)-1], 10)
	}

	return exist, nil
}

// QuerySrcFile returns file
func (bc *BaseContract) QuerySrcFile(name string) (string, error) {
	if bc.srcFs == nil {
		return "", fmt.Errorf("embed fs is nil")
	}

	b, err := bc.srcFs.ReadFile(name)
	return string(b), err
}

// QuerySrcPartFile returns part of file
// start - include
// end   - exclude
func (bc *BaseContract) QuerySrcPartFile(name string, start int, end int) (string, error) {
	if bc.srcFs == nil {
		return "", fmt.Errorf("embed fs is nil")
	}

	f, err := bc.srcFs.ReadFile(name)
	if err != nil {
		return "", err
	}

	if start < 0 {
		start = 0
	}

	if end < 0 {
		end = 0
	}

	if end > len(f) {
		end = len(f)
	}

	if start > end {
		return "", fmt.Errorf("start more then end")
	}

	return string(f[start:end]), nil
}

// QueryNameOfFiles returns list path/name of embed files
func (bc *BaseContract) QueryNameOfFiles() ([]string, error) {
	if bc.srcFs == nil {
		return nil, fmt.Errorf("embed fs is nil")
	}

	fs, err := bc.srcFs.ReadDir(".")
	if err != nil {
		return nil, err
	}

	res := make([]string, 0)
	for _, f := range fs {
		if f.IsDir() {
			r, e := bc.readDir(f.Name())
			if e != nil {
				return nil, e
			}
			res = append(res, r...)
			continue
		}
		res = append(res, f.Name())
	}
	return res, nil
}

func (bc *BaseContract) readDir(name string) ([]string, error) {
	fs, err := bc.srcFs.ReadDir(name)
	if err != nil {
		return nil, err
	}

	res := make([]string, 0)
	for _, f := range fs {
		if f.IsDir() {
			r, e := bc.readDir(name + "/" + f.Name())
			if e != nil {
				return nil, e
			}
			res = append(res, r...)
			continue
		}
		res = append(res, name+"/"+f.Name())
	}

	return res, nil
}

// QueryBuildInfo returns debug.BuildInfo struct with build information, stored in binary file or error if it is occurs
func (bc *BaseContract) QueryBuildInfo() (*debug.BuildInfo, error) {
	bi, err := version.BuildInfo()
	if err != nil {
		return nil, err
	}

	return bi, nil
}

// QueryCoreChaincodeIDName returns CORE_CHAINCODE_ID_NAME
func (bc *BaseContract) QueryCoreChaincodeIDName() (string, error) {
	res := version.CoreChaincodeIDName()
	return res, nil
}

// QuerySystemEnv returns system environment
func (bc *BaseContract) QuerySystemEnv() (map[string]string, error) {
	res := version.SystemEnv()
	return res, nil
}

// TxHealthCheck can be called by an administrator of the contract for checking if
// the business logic of the chaincode is still alive.
func (bc *BaseContract) TxHealthCheck(_ *types.Sender) error {
	return nil
}

// BaseContractInterface is a base contract interface for all contracts
type BaseContractInterface interface { //nolint:interfacebloat
	// WARNING!
	// Private interface methods can only be implemented in this package.
	// Bad practice. Can only be used to embed the necessary structure
	// and no more. Needs refactoring in the future.

	addMethod(string)
	baseContractInit(BaseContractInterface)
	setStubAndInitArgs(stub shim.ChaincodeStubInterface, atomyzeSKI []byte, args []string, noncePrefix StateKey)
	setSrcFs(*embed.FS)
	tokenBalanceAdd(address *types.Address, amount *big.Int, token string) error

	// ------------------------------------------------------------------

	GetStub() shim.ChaincodeStubInterface
	GetID() string

	TokenBalanceTransfer(from *types.Address, to *types.Address, amount *big.Int, reason string) error
	AllowedBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error

	TokenBalanceGet(address *types.Address) (*big.Int, error)
	TokenBalanceAdd(address *types.Address, amount *big.Int, reason string) error
	TokenBalanceSub(address *types.Address, amount *big.Int, reason string) error

	AllowedBalanceGet(token string, address *types.Address) (*big.Int, error)
	AllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error
	AllowedBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error

	AllowedBalanceGetAll(address *types.Address) (map[string]string, error)

	IndustrialBalanceGet(address *types.Address) (map[string]string, error)
	IndustrialBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error
	IndustrialBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error
	IndustrialBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error

	AllowedIndustrialBalanceAdd(address *types.Address, industrialAssets []*pb.Asset, reason string) error
	AllowedIndustrialBalanceSub(address *types.Address, industrialAssets []*pb.Asset, reason string) error
	AllowedIndustrialBalanceTransfer(from *types.Address, to *types.Address, industrialAssets []*pb.Asset, reason string) error
}
