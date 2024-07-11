package core

import (
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/routing"
	"github.com/anoideaopen/foundation/core/stringsx"
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
	envs           sync.Map // map[int64]*environment: (goid -> environment)
	srcFs          *embed.FS
	config         *pb.ContractConfig
	tracingHandler *telemetry.TracingHandler
	lockTH         sync.RWMutex
	isService      bool
	router         routing.Router
}

var _ BaseContractInterface = &BaseContract{}

type environment struct {
	stub  shim.ChaincodeStubInterface
	trace telemetry.TraceContext
}

func (bc *BaseContract) setEnv(env *environment) {
	bc.envs.Store(goid(), env)
}

func (bc *BaseContract) getEnv() *environment {
	if env, ok := bc.envs.Load(goid()); ok {
		return env.(*environment) //nolint:forcetypeassert
	}

	return nil
}

func (bc *BaseContract) delEnv() {
	bc.envs.Delete(goid())
}

func goid() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}

func (bc *BaseContract) setRouter(router routing.Router) {
	bc.router = router
}

func (bc *BaseContract) Router() routing.Router {
	return bc.router
}

func (bc *BaseContract) setSrcFs(srcFs *embed.FS) {
	bc.srcFs = srcFs
}

// GetStub returns stub
func (bc *BaseContract) GetStub() shim.ChaincodeStubInterface {
	if env := bc.getEnv(); env != nil {
		return env.stub
	}

	return nil
}

// GetMethods returns list of methods
func (bc *BaseContract) GetMethods(bci BaseContractInterface) []string {
	contractMethods := bci.Router().Methods()

	methods := make([]string, 0, len(contractMethods))
	for name, method := range contractMethods {
		if !bc.isMethodDisabled(method) {
			methods = append(methods, name)
		}
	}

	sort.Strings(methods)

	return methods
}

func (bc *BaseContract) isMethodDisabled(method routing.Method) bool {
	for _, disabled := range bc.ContractConfig().GetOptions().GetDisabledFunctions() {
		if method.Method == disabled {
			return true
		}
		if bc.ContractConfig().GetOptions().GetDisableSwaps() &&
			stringsx.OneOf(method.Method, "QuerySwapGet", "TxSwapBegin", "TxSwapCancel") {
			return true
		}
		if bc.ContractConfig().GetOptions().GetDisableMultiSwaps() &&
			stringsx.OneOf(method.Method, "QueryMultiSwapGet", "TxMultiSwapBegin", "TxMultiSwapCancel") {
			return true
		}
	}
	return false
}

func (bc *BaseContract) QueryGetNonce(owner *types.Address) (string, error) {
	prefix := hex.EncodeToString([]byte{StateKeyNonce})
	key, err := bc.GetStub().CreateCompositeKey(prefix, []string{owner.String()})
	if err != nil {
		return "", err
	}

	data, err := bc.GetStub().GetState(key)
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
	logger.Logger().Warning("HealthCheck", "txId", bc.GetStub().GetTxID())
	return nil
}

func (bc *BaseContract) GetID() string { // deprecated
	return bc.ContractConfig().GetSymbol()
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
	logger.Logger().Warning("HealthCheckNb", "txId", bc.GetStub().GetTxID())
	return nil
}

// GetTraceContext returns trace context. Using for call methods only
func (bc *BaseContract) GetTraceContext() telemetry.TraceContext {
	if env := bc.getEnv(); env != nil {
		return env.trace
	}

	return telemetry.TraceContext{}
}

// setTracingHandler sets base contract tracingHandler
func (bc *BaseContract) setTracingHandler(th *telemetry.TracingHandler) {
	bc.tracingHandler = th
}

// TracingHandler returns base contract tracingHandler
func (bc *BaseContract) TracingHandler() *telemetry.TracingHandler {
	var th *telemetry.TracingHandler

	// read
	bc.lockTH.RLock()
	if bc.tracingHandler != nil {
		th = bc.tracingHandler
	}
	bc.lockTH.RUnlock()

	if th != nil {
		return th
	}

	// set
	bc.lockTH.Lock()
	defer bc.lockTH.Unlock()

	if bc.tracingHandler != nil {
		return bc.tracingHandler
	}

	bc.setupTracing()

	return bc.tracingHandler
}

// setIsService sets base contract isService
func (bc *BaseContract) setIsService() {
	bc.isService = true
}

// IsService returns true if chaincode runs as a service
func (bc *BaseContract) IsService() bool {
	return bc.isService
}

// setupTracing lazy telemetry tracing setup.
func (bc *BaseContract) setupTracing() {
	serviceName := "chaincode-" + bc.GetID()

	// Check if the environment variable with the tracing collector endpoint exists
	endpointFromEnv, ok := os.LookupEnv(telemetry.TracingCollectorEndpointEnv)

	traceConfig := bc.ContractConfig().GetTracingCollectorEndpoint()

	// If the chaincode is not operating as a service or the environment variable with the endpoint
	// does not exist in the system, use the contract configuration for tracing.
	if bc.IsService() && ok {
		traceConfig = &pb.CollectorEndpoint{
			Endpoint:                 endpointFromEnv,
			AuthorizationHeaderKey:   os.Getenv(telemetry.TracingCollectorAuthHeaderKey),
			AuthorizationHeaderValue: os.Getenv(telemetry.TracingCollectorAuthHeaderValue),
			TlsCa:                    os.Getenv(telemetry.TracingCollectorCaPem),
		}
	}

	telemetry.InstallTraceProvider(traceConfig, serviceName)

	th := &telemetry.TracingHandler{}
	th.Tracer = otel.Tracer(serviceName)
	th.Propagators = otel.GetTextMapPropagator()
	th.TracingInit()

	bc.setTracingHandler(th)
}
