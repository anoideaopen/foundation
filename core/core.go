package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"embed"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime/debug"

	"github.com/anoideaopen/foundation/core/initialize"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

const (
	assertInterfaceErrMsg = "assertion interface -> error is failed"

	chaincodeExecModeEnv    = "CHAINCODE_EXEC_MODE"
	chaincodeExecModeServer = "server"
	chaincodeCcIDEnv        = "CHAINCODE_ID"

	chaincodeServerDefaultPort = "9999"
	chaincodeServerPortEnv     = "CHAINCODE_SERVER_PORT"
)

// NonceCheckFn is a function for checking nonce
type NonceCheckFn func(shim.ChaincodeStubInterface, *types.Sender, uint64) error

// ChaincodeOption func for each Opts argument
type ChaincodeOption func(opts *chaincodeOptions) error

// opts allows the user to specify more advanced options
type chaincodeOptions struct {
	SrcFs *embed.FS
}

// ChainCode is a chaincode	struct which implements shim.Chaincode interface
type ChainCode struct {
	contract          BaseContractInterface
	methods           map[string]*Fn
	disableSwaps      bool
	disableMultiSwaps bool
	txTTL             uint
	batchPrefix       string
	nonceTTL          uint
	noncePrefix       StateKey
	nonceCheckFn      NonceCheckFn
}

// WithSrcFS specifies a set src fs
func WithSrcFS(fs *embed.FS) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.SrcFs = fs
		return nil
	}
}

// NewCC creates new ChainCode
func NewCC(
	cc BaseContractInterface,
	options *ContractOptions,
	chOptions ...ChaincodeOption,
) (*ChainCode, error) {
	chOpts := chaincodeOptions{}
	for _, option := range chOptions {
		err := option(&chOpts)
		if err != nil {
			return &ChainCode{}, fmt.Errorf("failed to read opts: %w", err)
		}
	}

	cc.baseContractInit(cc)
	cc.setSrcFs(chOpts.SrcFs)

	methods, err := ParseContract(cc, options)
	if err != nil {
		return &ChainCode{}, err
	}

	out := &ChainCode{
		contract:     cc,
		methods:      methods,
		batchPrefix:  batchKey,
		noncePrefix:  StateKeyNonce,
		nonceCheckFn: checkNonce(0, StateKeyNonce),
	}

	if options != nil {
		out.disableSwaps = options.DisableSwaps
		out.disableMultiSwaps = options.DisableMultiSwaps
		out.txTTL = options.TxTTL
		if options.BatchPrefix != "" {
			out.batchPrefix = options.BatchPrefix
		}
		if options.NonceTTL != 0 {
			out.nonceTTL = options.NonceTTL
		}
		if options.IsOtherNoncePrefix {
			out.noncePrefix = StateKeyPassedNonce
		}

		out.nonceCheckFn = checkNonce(out.nonceTTL, out.noncePrefix)
	}

	return out, nil
}

// Init initializes chaincode
func (cc *ChainCode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	err := initialize.InitChaincode(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// Invoke invokes chaincode
func (cc *ChainCode) Invoke(stub shim.ChaincodeStubInterface) (r peer.Response) {
	r = shim.Error("panic invoke")
	defer func() {
		if rc := recover(); rc != nil {
			log.Printf("panic invoke\nrc: %v\nstack: %s\n", rc, debug.Stack())
		}
	}()

	err := cc.ValidateTxID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	creatorSKI, hashedCert, err := creatorSKIAndHashedCertByStub(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	functionName, args := stub.GetFunctionAndParameters()
	switch functionName {
	case "batchExecute":
		return cc.batchExecuteHandler(stub, creatorSKI, hashedCert, args)
	case "swapDone":
		return cc.swapDoneHandler(stub, args)
	case "multiSwapDone":
		return cc.multiSwapDoneHandler(stub, args)
	case "createCCTransferTo", "cancelCCTransferFrom", "commitCCTransferFrom",
		"deleteCCTransferFrom", "deleteCCTransferTo":
		initArgs, err := initialize.LoadInitArgs(stub)
		if err != nil {
			return shim.Error(fmt.Sprintf("incorrect tx id %s", err.Error()))
		}

		err = validateRobotSKI(initArgs.RobotSKI, creatorSKI, hashedCert)
		if err != nil {
			return shim.Error(err.Error())
		}
	}

	// fetch function details by function name
	fn, err := cc.FetchFnByName(functionName)
	if err != nil {
		return shim.Error(err.Error())
	}

	// handle invoke and query methods executed without batch process
	if fn.noBatch {
		return cc.noBatchHandler(stub, functionName, fn, args)
	}

	// handle invoke method with batch process
	return cc.BatchHandler(stub, functionName, fn, args)
}

func validateRobotSKI(robotSKI []byte, creatorSKI [32]byte, hashedCert [32]byte) error {
	if !bytes.Equal(hashedCert[:], robotSKI) &&
		!bytes.Equal(creatorSKI[:], robotSKI) {
		return fmt.Errorf("unauthorized: robotSKI and creatorSKI, hashedCert is not equal")
	}
	return nil
}

// ValidateTxID validates tx id
func (cc *ChainCode) ValidateTxID(stub shim.ChaincodeStubInterface) error {
	_, err := hex.DecodeString(stub.GetTxID())
	if err != nil {
		return fmt.Errorf("incorrect tx id: %w", err)
	}
	return nil
}

// FetchFnByName fetches function by name
func (cc *ChainCode) FetchFnByName(f string) (*Fn, error) {
	method, exists := cc.methods[f]
	if !exists {
		return nil, fmt.Errorf("method not found: '%s'", f)
	}
	return method, nil
}

// BatchHandler handles batch process
func (cc *ChainCode) BatchHandler(stub shim.ChaincodeStubInterface, funcName string, fn *Fn, args []string) peer.Response {
	sender, args, nonce, err := cc.checkAuthIfNeeds(stub, fn, funcName, args, true)
	if err != nil {
		return shim.Error(err.Error())
	}
	args, err = doPrepareToSave(stub, fn, args)
	if err != nil {
		return shim.Error(err.Error())
	}

	if err = cc.saveToBatch(stub, funcName, sender, args[:len(fn.in)], nonce); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (cc *ChainCode) noBatchHandler(stub shim.ChaincodeStubInterface, funcName string, fn *Fn, args []string) peer.Response {
	if fn.query {
		stub = newQueryStub(stub)
	}

	sender, args, _, err := cc.checkAuthIfNeeds(stub, fn, funcName, args, true)
	if err != nil {
		return shim.Error(err.Error())
	}
	args, err = doPrepareToSave(stub, fn, args)
	if err != nil {
		return shim.Error(err.Error())
	}

	initArgs, err := initialize.LoadInitArgs(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("incorrect tx id %s", err.Error()))
	}
	resp, err := cc.callMethod(stub, fn, sender, args, initArgs.PlatformSKI, initArgs.Args)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(resp)
}

func creatorSKIAndHashedCertByStub(stub shim.ChaincodeStubInterface) ([32]byte, [32]byte, error) {
	var creatorSKI [32]byte
	var hashedCert [32]byte
	creator, err := stub.GetCreator()
	if err != nil {
		return creatorSKI, hashedCert, err
	}

	var identity msp.SerializedIdentity
	if err = pb.Unmarshal(creator, &identity); err != nil {
		return creatorSKI, hashedCert, err
	}

	b, _ := pem.Decode(identity.IdBytes)
	parsed, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return creatorSKI, hashedCert, err
	}

	pk, ok := parsed.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return creatorSKI, hashedCert, errors.New("public key type assertion failed")
	}

	creatorSKI = sha256.Sum256(elliptic.Marshal(pk.Curve, pk.X, pk.Y))
	hashedCert = sha3.Sum256(creator)

	return creatorSKI, hashedCert, nil
}

func (cc *ChainCode) multiSwapDoneHandler(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if cc.disableMultiSwaps {
		return shim.Error("multiswaps disabled")
	}
	initArgs, err := initialize.LoadInitArgs(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("incorrect tx id %s", err.Error()))
	}
	_, contract := copyContract(cc.contract, stub, initArgs.PlatformSKI, initArgs.Args, cc.noncePrefix)
	return multiSwapUserDone(contract, args[0], args[1])
}

func (cc *ChainCode) swapDoneHandler(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if cc.disableSwaps {
		return shim.Error("swaps disabled")
	}
	initArgs, err := initialize.LoadInitArgs(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("incorrect tx id %s", err.Error()))
	}
	_, contract := copyContract(cc.contract, stub, initArgs.PlatformSKI, initArgs.Args, cc.noncePrefix)
	return swapUserDone(contract, args[0], args[1])
}

func (cc *ChainCode) batchExecuteHandler(stub shim.ChaincodeStubInterface, creatorSKI [32]byte, hashedCert [32]byte, args []string) peer.Response {
	initArgs, err := initialize.LoadInitArgs(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("incorrect tx id %s", err.Error()))
	}

	err = validateRobotSKI(initArgs.RobotSKI, creatorSKI, hashedCert)
	if err != nil {
		return shim.Error(err.Error())
	}

	return cc.batchExecute(stub, args[0], initArgs.PlatformSKI, initArgs.Args)
}

func (cc *ChainCode) callMethod(
	stub shim.ChaincodeStubInterface,
	method *Fn,
	sender *proto.Address,
	args []string,
	platformSKI []byte,
	initArgs []string,
) ([]byte, error) {
	values, err := doConvertToCall(stub, method, args)
	if err != nil {
		return nil, err
	}
	if sender != nil {
		values = append([]reflect.Value{
			reflect.ValueOf(types.NewSenderFromAddr((*types.Address)(sender))),
		}, values...)
	}

	contract, _ := copyContract(cc.contract, stub, platformSKI, initArgs, cc.noncePrefix)

	out := method.fn.Call(append([]reflect.Value{contract}, values...))
	errInt := out[0].Interface()
	if method.out {
		errInt = out[1].Interface()
	}
	if errInt != nil {
		err, ok := errInt.(error)
		if !ok {
			return nil, errors.New(assertInterfaceErrMsg)
		}
		return nil, err
	}

	if method.out {
		return json.Marshal(out[0].Interface())
	}
	return nil, nil
}

func doConvertToCall(stub shim.ChaincodeStubInterface, method *Fn, args []string) ([]reflect.Value, error) {
	found := len(args)
	expected := len(method.in)
	if found < expected {
		return nil, fmt.Errorf("incorrect number of arguments, found %d but expected more than %d", found, expected)
	}

	vArgs := make([]reflect.Value, len(method.in))
	for i := range method.in {
		var impl reflect.Value
		if method.in[i].kind.Kind().String() == "ptr" {
			impl = reflect.New(method.in[i].kind.Elem())
		} else {
			impl = reflect.New(method.in[i].kind).Elem()
		}

		res := method.in[i].convertToCall.Call([]reflect.Value{
			impl,
			reflect.ValueOf(stub), reflect.ValueOf(args[i]),
		})

		if res[1].Interface() != nil {
			err, ok := res[1].Interface().(error)
			if !ok {
				return nil, errors.New(assertInterfaceErrMsg)
			}
			return nil, err
		}
		vArgs[i] = res[0]
	}
	return vArgs, nil
}

func doPrepareToSave(stub shim.ChaincodeStubInterface, method *Fn, args []string) ([]string, error) {
	if len(args) < len(method.in) {
		return nil, fmt.Errorf("incorrect number of arguments. current count of args is %d but expected more than %d",
			len(args), len(method.in))
	}
	as := make([]string, len(method.in))
	for i := range method.in {
		var impl reflect.Value
		if method.in[i].kind.Kind().String() == "ptr" {
			impl = reflect.New(method.in[i].kind.Elem())
		} else {
			impl = reflect.New(method.in[i].kind).Elem()
		}

		var ok bool
		if method.in[i].prepareToSave.IsValid() {
			res := method.in[i].prepareToSave.Call([]reflect.Value{
				impl,
				reflect.ValueOf(stub), reflect.ValueOf(args[i]),
			})
			if res[1].Interface() != nil {
				err, ok := res[1].Interface().(error)
				if !ok {
					return nil, errors.New(assertInterfaceErrMsg)
				}
				return nil, err
			}
			as[i], ok = res[0].Interface().(string)
			if !ok {
				return nil, errors.New(assertInterfaceErrMsg)
			}
			continue
		}

		// if method PrepareToSave don't have exists
		// use ConvertToCall to check converting
		res := method.in[i].convertToCall.Call([]reflect.Value{
			impl,
			reflect.ValueOf(stub), reflect.ValueOf(args[i]),
		})
		if res[1].Interface() != nil {
			err, ok := res[1].Interface().(error)
			if !ok {
				return nil, errors.New(assertInterfaceErrMsg)
			}
			return nil, err
		}

		as[i] = args[i] // in this case we don't convert argument
	}
	return as, nil
}

func copyContract(
	orig BaseContractInterface,
	stub shim.ChaincodeStubInterface,
	platformSKI []byte,
	initArgs []string,
	noncePrefix StateKey,
) (reflect.Value, BaseContractInterface) {
	cp := reflect.New(reflect.ValueOf(orig).Elem().Type())
	val := reflect.ValueOf(orig).Elem()
	for i := 0; i < val.NumField(); i++ {
		if cp.Elem().Field(i).CanSet() {
			cp.Elem().Field(i).Set(val.Field(i))
		}
	}
	contract, ok := cp.Interface().(BaseContractInterface)
	if !ok {
		return cp, nil
	}
	contract.setStubAndInitArgs(stub, platformSKI, initArgs, noncePrefix)
	return cp, contract
}

// Start starts chaincode
func (cc *ChainCode) Start() error {
	// get chaincode execution mode
	execMode := os.Getenv(chaincodeExecModeEnv)
	// if exec mode is not chaincode-as-server or not defined start chaincode as usual
	if execMode != chaincodeExecModeServer {
		return shim.Start(cc)
	}
	// if chaincode exec mode is chaincode-as-server we should propagate variables
	var ccID string
	// if chaincode was set during runtime build, use it
	if ccID = os.Getenv(chaincodeCcIDEnv); ccID == "" {
		return fmt.Errorf("need to specify chaincode id if running as server")
	}

	port := os.Getenv(chaincodeServerPortEnv)
	if port == "" {
		port = chaincodeServerDefaultPort
	}

	srv := shim.ChaincodeServer{
		CCID:    ccID,
		Address: fmt.Sprintf("%s:%s", "0.0.0.0", port),
		CC:      cc,
		TLSProps: shim.TLSProperties{
			Disabled: true,
		},
	}
	return srv.Start()
}
