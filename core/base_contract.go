package core

import (
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime/debug"
	"sort"
	"strconv"

	"github.com/anoideaopen/foundation/core/contract"
	"github.com/anoideaopen/foundation/core/reflectx"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/version"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"go.opentelemetry.io/otel"
	"google.golang.org/protobuf/encoding/protojson"
)

// BaseContract is a base contract for all contracts
type BaseContract struct {
	stub           shim.ChaincodeStubInterface
	noncePrefix    byte
	srcFs          *embed.FS
	config         *pb.ContractConfig
	traceCtx       telemetry.TraceContext
	tracingHandler *telemetry.TracingHandler
}

var _ BaseContractInterface = &BaseContract{}

func (bc *BaseContract) setSrcFs(srcFs *embed.FS) {
	bc.srcFs = srcFs
}

// GetStub returns stub
func (bc *BaseContract) GetStub() shim.ChaincodeStubInterface {
	return bc.stub
}

// GetMethods returns list of methods
func (bc *BaseContract) GetMethods(bci BaseContractInterface) []string {
	router, err := buildRouter(bci)
	if err != nil {
		panic(err)
	}

	contractMethods := router.Methods()

	methods := make([]string, 0, len(contractMethods))
	for name := range contractMethods {
		methods = append(methods, name)
	}

	sort.Strings(methods)

	return methods
}

func (bc *BaseContract) SetStub(stub shim.ChaincodeStubInterface) {
	bc.stub = stub
	bc.noncePrefix = StateKeyNonce
}

func (bc *BaseContract) QueryGetNonce(owner *types.Address) (string, error) {
	prefix := hex.EncodeToString([]byte{bc.noncePrefix})
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
		exist = strconv.FormatUint(lastNonce.GetNonce()[len(lastNonce.GetNonce())-1], 10)
	}

	return exist, nil
}

// QuerySrcFile returns file
func (bc *BaseContract) QuerySrcFile(name string) (string, error) {
	if bc.srcFs == nil {
		return "", errors.New("embed fs is nil")
	}

	b, err := bc.srcFs.ReadFile(name)
	return string(b), err
}

// QuerySrcPartFile returns part of file
// start - include
// end   - exclude
func (bc *BaseContract) QuerySrcPartFile(name string, start int, end int) (string, error) {
	if bc.srcFs == nil {
		return "", errors.New("embed fs is nil")
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
		return "", errors.New("start more then end")
	}

	return string(f[start:end]), nil
}

// QueryNameOfFiles returns list path/name of embed files
func (bc *BaseContract) QueryNameOfFiles() ([]string, error) {
	if bc.srcFs == nil {
		return nil, errors.New("embed fs is nil")
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

func (bc *BaseContract) ID() string {
	return bc.config.GetSymbol()
}

func (bc *BaseContract) GetID() string { // deprecated
	return bc.ID()
}

func (bc *BaseContract) ValidateConfig(config []byte) error {
	var cfg pb.Config
	if err := protojson.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("unmarshalling base config data failed: %w", err)
	}

	if cfg.GetContract() == nil {
		return errors.New("validating contract config: contract config is not set or broken")
	}

	if err := cfg.GetContract().ValidateAll(); err != nil {
		return fmt.Errorf("validating contract config: %w", err)
	}

	return nil
}

func (bc *BaseContract) ApplyContractConfig(config *pb.ContractConfig) error {
	bc.config = config

	return nil
}

func (bc *BaseContract) ContractConfig() *pb.ContractConfig {
	return bc.config
}

// NBTxHealthCheckNb - the same but not batched
func (bc *BaseContract) NBTxHealthCheckNb(_ *types.Sender) error {
	return nil
}

// setTraceContext sets context for telemetry. For call methods only
func (bc *BaseContract) setTraceContext(traceCtx telemetry.TraceContext) {
	bc.traceCtx = traceCtx
}

// GetTraceContext returns trace context. Using for call methods only
func (bc *BaseContract) GetTraceContext() telemetry.TraceContext {
	return bc.traceCtx
}

// setTracingHandler sets base contract tracingHandler
func (bc *BaseContract) setTracingHandler(th *telemetry.TracingHandler) {
	bc.tracingHandler = th
}

// TracingHandler returns base contract tracingHandler
func (bc *BaseContract) TracingHandler() *telemetry.TracingHandler {
	if bc.tracingHandler == nil {
		bc.setupTracing()
	}

	return bc.tracingHandler
}

// setupTracing lazy telemetry tracing setup.
func (bc *BaseContract) setupTracing() {
	serviceName := "chaincode-" + bc.GetID()

	telemetry.InstallTraceProvider(bc.ContractConfig().GetTracingCollectorEndpoint(), serviceName)

	th := &telemetry.TracingHandler{}
	th.Tracer = otel.Tracer(serviceName)
	th.Propagators = otel.GetTextMapPropagator()
	th.TracingInit()

	bc.setTracingHandler(th)
}

func buildRouter(in contract.Base) (contract.Router, error) {
	if router, ok := in.(contract.Router); ok {
		return router, nil
	}

	return reflectx.NewRouter(in)
}

// BaseContractInterface represents BaseContract interface
type BaseContractInterface interface { //nolint:interfacebloat
	contract.Base

	// WARNING!
	// Private interface methods can only be implemented in this package.
	// Bad practice. Can only be used to embed the necessary structure
	// and no more. Needs refactoring in the future.

	setSrcFs(*embed.FS)
	tokenBalanceAdd(address *types.Address, amount *big.Int, token string) error

	// ------------------------------------------------------------------
	GetID() string

	TokenBalanceTransfer(from *types.Address, to *types.Address, amount *big.Int, reason string) error
	AllowedBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error

	TokenBalanceGet(address *types.Address) (*big.Int, error)
	TokenBalanceAdd(address *types.Address, amount *big.Int, reason string) error
	TokenBalanceSub(address *types.Address, amount *big.Int, reason string) error

	TokenBalanceAddWithTicker(address *types.Address, amount *big.Int, ticker string, reason string) error
	TokenBalanceSubWithTicker(address *types.Address, amount *big.Int, ticker string, reason string) error

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

	setTraceContext(traceCtx telemetry.TraceContext)
	GetTraceContext() telemetry.TraceContext

	setTracingHandler(th *telemetry.TracingHandler)
	TracingHandler() *telemetry.TracingHandler
}
