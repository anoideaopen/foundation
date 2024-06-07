package core

import (
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/contract"
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/reflectx"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/hlfcreator"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// requireInterfaceErrMsg is the error message used when an interface to error type requireion fails.
	requireInterfaceErrMsg = "requireion interface -> error is failed"

	// chaincodeExecModeEnv is the environment variable that specifies the execution mode of the chaincode.
	chaincodeExecModeEnv = "CHAINCODE_EXEC_MODE"
	// chaincodeExecModeServer is the value that, when set for the CHAINCODE_EXEC_MODE environment variable,
	// indicates that the chaincode is running in server mode.
	chaincodeExecModeServer = "server"
	// chaincodeCcIDEnv is the environment variable that holds the chaincode ID.
	chaincodeCcIDEnv = "CHAINCODE_ID"

	// chaincodeServerDefaultPort is the default port on which the chaincode server listens if no other port is specified.
	chaincodeServerDefaultPort = "9999"
	// chaincodeServerPortEnv is the environment variable that specifies the port on which the chaincode server listens.
	chaincodeServerPortEnv = "CHAINCODE_SERVER_PORT"

	// TLS environment variables for the chaincode's TLS configuration with files.
	// tlsKeyFileEnv is the environment variable that specifies the private key file for TLS communication.
	tlsKeyFileEnv = "CHAINCODE_TLS_KEY_FILE"
	// tlsCertFileEnv is the environment variable that specifies the public key certificate file for TLS communication.
	tlsCertFileEnv = "CHAINCODE_TLS_CERT_FILE"
	// tlsClientCACertsFileEnv is the environment variable that specifies the client CA certificates file for TLS communication.
	tlsClientCACertsFileEnv = "CHAINCODE_TLS_CLIENT_CA_CERTS_FILE"

	// TLS environment variables for the chaincode's TLS configuration, directly from ENVs.
	// tlsKeyEnv is the environment variable that specifies the private key for TLS communication.
	tlsKeyEnv = "CHAINCODE_TLS_KEY"
	// tlsCertEnv is the environment variable that specifies the public key certificate for TLS communication.
	tlsCertEnv = "CHAINCODE_TLS_CERT"
	// tlsClientCACertsEnv is the environment variable that specifies the client CA certificates for TLS communication.
	tlsClientCACertsEnv = "CHAINCODE_TLS_CLIENT_CA_CERTS"
)

var (
	ErrSwapDisabled      = errors.New("swap is disabled")
	ErrMultiSwapDisabled = errors.New("multi-swap is disabled")
)

const (
	BatchExecute         = "batchExecute"
	SwapDone             = "swapDone"
	MultiSwapDone        = "multiSwapDone"
	CreateCCTransferTo   = "createCCTransferTo"
	DeleteCCTransferTo   = "deleteCCTransferTo"
	CommitCCTransferFrom = "commitCCTransferFrom"
	CancelCCTransferFrom = "cancelCCTransferFrom"
	DeleteCCTransferFrom = "deleteCCTransferFrom"
	CreateIndex          = "createIndex"
	ExecuteTasks         = "executeTasks"
)

// ChaincodeOption represents a function that applies configuration options to
// a chaincodeOptions object.
//
// opts: A pointer to a chaincodeOptions object that the function will modify.
//
// error: The function returns an error if applying the option fails.
type ChaincodeOption func(opts *chaincodeOptions) error

// TLS holds the key and certificate data for TLS communication, as well as
// client CA certificates for peer verification if needed.
type TLS struct {
	Key           []byte // Private key for TLS authentication.
	Cert          []byte // Public certificate for TLS authentication.
	ClientCACerts []byte // Optional client CA certificates for verifying connecting peers.
}

// chaincodeOptions is a structure that holds advanced options for configuring
// a ChainCode instance.
type chaincodeOptions struct {
	SrcFS        *embed.FS             // SrcFS is a file system that contains the source files for the chaincode.
	TLS          *TLS                  // TLS contains the TLS configuration for the chaincode.
	ConfigMapper contract.ConfigMapper // ConfigMapper maps the arguments to a proto.Config instance.
}

// Chaincode defines the structure for a chaincode instance, with methods,
// configuration, and options for transaction processing.
type Chaincode struct {
	contract     BaseContractInterface // Contract interface containing the chaincode logic.
	tls          shim.TLSProperties    // TLS configuration properties.
	router       contract.Router       // Router for routing contract calls.
	configMapper contract.ConfigMapper // ConfigMapper maps the arguments to a proto.Config instance.
}

func (cc *Chaincode) Router() contract.Router {
	if cc.router != nil {
		return cc.router
	}

	if router, ok := cc.contract.(contract.Router); ok {
		cc.router = router
		return cc.router
	}

	// Error is checked in Init.
	cc.router, _ = reflectx.NewRouter(cc.contract, reflectx.RouterConfig{
		SwapsDisabled:      cc.contract.ContractConfig().GetOptions().GetDisableSwaps(),
		MultiSwapsDisabled: cc.contract.ContractConfig().GetOptions().GetDisableMultiSwaps(),
		DisabledMethods:    cc.contract.ContractConfig().GetOptions().GetDisabledFunctions(),
	})

	return cc.router
}

func (cc *Chaincode) Method(functionName string) (contract.Method, error) {
	if method, ok := cc.Router().Methods()[functionName]; ok {
		return method, nil
	}

	return contract.Method{}, fmt.Errorf("method '%s' not found", functionName)
}

// WithConfigMapper is a ChaincodeOption that specifies the ConfigMapper for the ChainCode.
//
// cm: An instance of the ConfigMapper interface.
//
// It returns a ChaincodeOption that sets the ConfigMapper field in the chaincodeOptions.
//
// Example:
//
//	configMapper := myCustomConfigMapper{}
//	chaincode := core.NewCC(cc, core.WithConfigMapper(configMapper))
func WithConfigMapper(cm contract.ConfigMapper) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.ConfigMapper = cm
		return nil
	}
}

// WithConfigMapperFunc is a ChaincodeOption that specifies the ConfigMapper for the ChainCode.
//
// cmf: A function implementing the ConfigMapper interface.
//
// It returns a ChaincodeOption that sets the ConfigMapper field in the chaincodeOptions.
//
// Example using FromArgsWithAdmin:
//
//	chaincode := core.NewCC(cc, core.WithConfigMapperFunc(func(args []string) (*proto.Config, error) {
//	    return config.FromArgsWithAdmin("ndm", args)
//	}))
//
// Example with manual mapping:
//
//	chaincode := core.NewCC(cc, core.WithConfigMapperFunc(func(args []string) (*proto.Config, error) {
//	    const requiredArgsCount = 4
//	    if len(args) != requiredArgsCount {
//	        return nil, fmt.Errorf("required args length is '%d', passed %d", requiredArgsCount, len(args))
//	    }
//	    robotSKI := args[1]
//	    if robotSKI == "" {
//	        return nil, fmt.Errorf("robot ski is empty")
//	    }
//	    issuerAddress := args[2]
//	    if issuerAddress == "" {
//	        return nil, fmt.Errorf("issuer address is empty")
//	    }
//	    adminAddress := args[3]
//	    if adminAddress == "" {
//	        return nil, fmt.Errorf("admin address is empty")
//	    }
//	    return &proto.Config{
//	        Contract: &proto.ContractConfig{
//	            Symbol: "TT",
//	            Admin:  &proto.Wallet{Address: adminAddress},
//	            RobotSKI: robotSKI,
//	        },
//	        Token: &proto.TokenConfig{
//	            Name: "Test Token",
//	            Issuer: &proto.Wallet{Address: issuerAddress},
//	        },
//	    }, nil
//	}))
func WithConfigMapperFunc(cmf contract.ConfigMapperFunc) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.ConfigMapper = cmf
		return nil
	}
}

// WithSrcFS is a ChaincodeOption that specifies the source file system to be used by the ChainCode.
//
// fs: A pointer to an embedded file system containing the chaincode files.
//
// It returns a ChaincodeOption that sets the SrcFs field in the chaincodeOptions.
func WithSrcFS(fs *embed.FS) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.SrcFS = fs
		return nil
	}
}

// WithTLS is a ChaincodeOption that specifies the TLS configuration for the ChainCode.
//
// tls: A pointer to a TLS structure containing the TLS certificates and keys.
//
// It returns a ChaincodeOption that sets the TLS field in the chaincodeOptions.
func WithTLS(tls *TLS) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.TLS = tls
		return nil
	}
}

// WithTLSFromFiles returns a ChaincodeOption that sets the TLS configuration
// for the ChainCode from provided file paths. It reads the specified files
// and uses their contents to configure TLS for the chaincode.
//
// keyPath: A string representing the file path to the TLS private key.
//
// certPath: A string representing the file path to the TLS public certificate.
//
// clientCACertPath: An optional string representing the file path to the client
// CA certificate. If no client CA certificate is needed, this can be left empty.
//
// It returns a ChaincodeOption or an error if reading any of the files fails.
//
// Example:
//
//	tlsOpt, err := core.WithTLSFromFiles("tls/key.pem", "tls/cert.pem", "tls/ca.pem")
//	if err != nil {
//	    log.Fatalf("Error configuring TLS: %v", err)
//	}
//	cc, err := core.NewCC(contractInstance, contractOptions, tlsOpt)
//	if err != nil {
//	    log.Fatalf("Error creating new chaincode instance: %v", err)
//	}
//
// This example sets up the chaincode TLS configuration using the key, certificate,
// and CA certificate files located in the "tls" directory. After obtaining the
// ChaincodeOption from WithTLSFromFiles, it is passed to NewCC to create a new
// instance of ChainCode with TLS enabled.
func WithTLSFromFiles(keyPath, certPath, clientCACertPath string) (ChaincodeOption, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, errors.New("failed to read TLS key: " + err.Error())
	}

	cert, err := os.ReadFile(certPath)
	if err != nil {
		return nil, errors.New("failed to read TLS certificate: " + err.Error())
	}

	tls := &TLS{
		Key:  key,
		Cert: cert,
	}

	if clientCACertPath != "" {
		clientCACerts, err := os.ReadFile(clientCACertPath)
		if err != nil {
			return nil, errors.New("failed to read client CA certificates: " + err.Error())
		}
		tls.ClientCACerts = clientCACerts
	}

	return func(o *chaincodeOptions) error {
		o.TLS = tls
		return nil
	}, nil
}

// NewCC creates a new instance of ChainCode with the given contract interface
// and configurable options. It initializes the ChainCode instance with the provided
// BaseContractInterface and applies advanced configuration settings through
// a combination of ChaincodeOption functions and environmental variables.
//
// The environmental variables are checked first to configure TLS settings,
// which takes precedence over the settings provided by the ChaincodeOption functions.
// The function will configure TLS if the respective environment variables contain
// the necessary information. These variables are:
//
// - CHAINCODE_TLS_KEY or CHAINCODE_TLS_KEY_FILE: For the private key in PEM format or file path.
// - CHAINCODE_TLS_CERT or CHAINCODE_TLS_CERT_FILE: For the public key certificate in PEM format or file path.
// - CHAINCODE_TLS_CLIENT_CA_CERTS or CHAINCODE_TLS_CLIENT_CA_CERTS_FILE: For the client CA certificates in PEM format or file path.
//
// If the environment variables do not provide the TLS configuration, the function
// will fall back to the configuration provided by ChaincodeOption functions, such as
// WithTLS or WithTLSFromFiles. If neither are provided, the TLS feature will remain
// disabled in the chaincode configuration.
//
// Args:
// cc: The BaseContractInterface which encapsulates the contract logic that
// the ChainCode will execute.
//
// options: ContractOptions is a pointer to the configuration settings that will
// be applied to the chaincode. The settings within options allow for fine-tuned
// control of the chaincode's behavior, such as transaction TTL, batching prefixes,
// and swap behavior. If this parameter is not needed, it can be omitted or set to nil.
//
// chOptions: A variadic number of ChaincodeOption function types which are used
// to apply specific configurations to the chaincodeOptions structure. These options
// may include configurations that can be overridden by environmental variables,
// particularly for TLS.
//
// Returns:
// A pointer to a ChainCode instance and an error. An error is non-nil
// if there is a failure in applying the provided ChaincodeOption functions, or if
// there is an issue with reading and processing the environmental variables for the
// TLS configuration.
//
// Example usage:
//
//	tlsConfig := &core.TLS{ /* ... */ }
//	cc, err := core.NewCC(contract, contractOptions, core.WithTLS(tlsConfig))
//	if err != nil {
//		// Handle error
//	}
//
// In the above example, tlsConfig provided by WithTLS will be overridden if the
// corresponding environmental variables for TLS configuration are set.
func NewCC(
	cc BaseContractInterface,
	chOptions ...ChaincodeOption,
) (*Chaincode, error) {
	empty := new(Chaincode) // Empty chaincode result fixes integration tests.

	// Default TLS properties, disabled unless keys and certs are provided.
	tlsProps := shim.TLSProperties{
		Disabled: true,
	}

	// Try to read TLS configuration from environment variables.
	key, cert, clientCACerts, err := readTLSConfigFromEnv()
	if err != nil {
		return empty, fmt.Errorf("error reading TLS config from environment: %w", err)
	}

	// If TLS configuration is found in environment variables, use it.
	if key != nil && cert != nil {
		tlsProps.Disabled = false
		tlsProps.Key = key
		tlsProps.Cert = cert
		tlsProps.ClientCACerts = clientCACerts
	}

	// Apply chaincode options provided by the caller.
	chOpts := chaincodeOptions{}
	for _, option := range chOptions {
		if option == nil {
			continue
		}
		err = option(&chOpts)
		if err != nil {
			return empty, fmt.Errorf("reading opts: %w", err)
		}
	}

	// If TLS was provided via options, overwrite env vars.
	if chOpts.TLS != nil {
		tlsProps.Disabled = false
		tlsProps.Key = chOpts.TLS.Key
		tlsProps.Cert = chOpts.TLS.Cert
		tlsProps.ClientCACerts = chOpts.TLS.ClientCACerts
	}

	// Initialize the contract.
	cc.setSrcFs(chOpts.SrcFS)

	// Set up the ChainCode structure.
	out := &Chaincode{
		contract:     cc,
		tls:          tlsProps,
		configMapper: chOpts.ConfigMapper,
	}

	return out, nil
}

// Init is called during chaincode instantiation to initialize any
// data. Note that upgrade also calls this function to reset or to migrate data.
//
// Args:
// stub: The shim.ChaincodeStubInterface containing the context of the call.
//
// Returns:
// - A success response if initialization succeeds.
// - An error response if it fails to get the creator or to initialize the chaincode.
func (cc *Chaincode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	creator, err := stub.GetCreator()
	if err != nil {
		return shim.Error("init: getting creator of transaction: " + err.Error())
	}
	if err = hlfcreator.ValidateAdminCreator(creator); err != nil {
		return shim.Error("init: validating admin creator: " + err.Error())
	}

	args := stub.GetStringArgs()

	var cfgBytes []byte
	switch {
	case config.IsJSON(args):
		cfgBytes = []byte(args[0])

	case cc.configMapper != nil:
		cfg, err := cc.configMapper.MapConfig(args)
		if err != nil {
			return shim.Error("init: mapping config: " + err.Error())
		}

		cfgBytes, err = protojson.Marshal(cfg)
		if err != nil {
			return shim.Error("init: marshaling config: " + err.Error())
		}

	default:
		// Handle args as positional parameters and fill the config structure.
		// TODO: Remove this code when all users have moved to JSON-config initialization.
		cfgBytes, err = config.FromInitArgs(stub.GetChannelID(), args) //nolint:staticcheck
		if err != nil {
			return shim.Error(fmt.Sprintf("init: parsing args old way: %s", err))
		}
	}

	if err = contract.ValidateConfig(cc.contract, cfgBytes); err != nil {
		return shim.Error("init: validating config: " + err.Error())
	}

	if err = config.Save(stub, cfgBytes); err != nil {
		return shim.Error("init: saving config: " + err.Error())
	}

	// Check if the contract implements the Router interface.
	if _, ok := cc.contract.(contract.Router); ok {
		return shim.Success(nil)
	}

	// If the contract does not implement the Router interface, check if it
	// is possible to create a router based on reflection.
	cfg, err := config.FromBytes(cfgBytes)
	if err != nil {
		return shim.Error("init: unmarshalling config: " + err.Error())
	}

	// Check for duplicate methods.
	if _, err = reflectx.NewRouter(cc.contract, reflectx.RouterConfig{
		SwapsDisabled:      cfg.GetContract().GetOptions().GetDisableSwaps(),
		MultiSwapsDisabled: cfg.GetContract().GetOptions().GetDisableMultiSwaps(),
		DisabledMethods:    cfg.GetContract().GetOptions().GetDisabledFunctions(),
	}); err != nil {
		return shim.Error("init: validating contract methods: " + err.Error())
	}

	return shim.Success(nil)
}

// Invoke is called to update or query the ledger in a proposal transaction.
// Given the function name, it delegates the execution to the respective handler.
//
// Args:
// stub: The shim.ChaincodeStubInterface containing the context of the call.
//
// Returns:
// - A response from the executed handler.
// - An error response if any validations fail or the required method is not found.
func (cc *Chaincode) Invoke(stub shim.ChaincodeStubInterface) (r peer.Response) {
	r = shim.Error("panic invoke")
	defer func() {
		if rc := recover(); rc != nil {
			log.Printf("panic invoke\nrc: %v\nstack: %s\n", rc, debug.Stack())
		}
	}()

	start := time.Now()

	// getting contract config
	cfgBytes, err := config.Load(stub)
	if err != nil {
		return shim.Error("invoke: loading raw config: " + err.Error())
	}

	// Apply config on all layers: base contract (SKI's & chaincode options),
	// token base attributes and extended token parameters.
	if err = contract.Configure(cc.contract, stub, cfgBytes); err != nil {
		return shim.Error("applying configutarion: " + err.Error())
	}

	// Getting carrier from transient map and creating tracing span
	traceCtx := cc.contract.TracingHandler().ContextFromStub(stub)
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "cc.Invoke")

	// Transaction context.
	span.AddEvent("get transactionID")
	transactionID := stub.GetTxID()

	span.SetAttributes(attribute.String("channel", stub.GetChannelID()))
	span.SetAttributes(attribute.String("tx_id", transactionID))
	span.SetAttributes(telemetry.MethodType(telemetry.MethodTx))

	span.AddEvent("get function and parameters")
	functionName, arguments := stub.GetFunctionAndParameters()

	span.AddEvent(fmt.Sprintf("begin id: %s, name: %s", transactionID, functionName))
	defer func() {
		span.AddEvent(fmt.Sprintf("end id: %s, name: %s, elapsed time %d ms",
			transactionID,
			functionName,
			time.Since(start).Milliseconds(),
		))

		span.End()
	}()

	span.AddEvent("validating transaction ID")
	if err = cc.ValidateTxID(stub); err != nil {
		errMsg := "invoke: validating transaction ID: " + err.Error()
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	span.AddEvent("getting creator")
	creatorBytes, err := stub.GetCreator()
	if err != nil {
		errMsg := "invoke: failed to get creator of transaction: " + err.Error()
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	span.AddEvent("getting creator SKI and hashed cert")
	creatorSKI, hashedCert, err := hlfcreator.CreatorSKIAndHashedCert(creatorBytes)
	if err != nil {
		errMsg := "invoke: validating creator: " + err.Error()
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	// it is probably worth checking if the function is not locked before it is executed.
	// You should also check with swap and multiswap locking and
	// display the error explicitly instead of saying that the function was not found.
	span.SetAttributes(attribute.String("method", functionName))
	switch functionName {
	case CreateIndex: // Creating a reverse index to find token owners.
		return cc.createIndexHandler(traceCtx, stub, arguments)

	case BatchExecute:
		return cc.batchExecuteHandler(traceCtx, stub, creatorSKI, hashedCert, arguments, cfgBytes)

	case SwapDone:
		return cc.swapDoneHandler(arguments)

	case MultiSwapDone:
		return cc.multiSwapDoneHandler(arguments)

	case CreateCCTransferTo,
		DeleteCCTransferTo,
		CommitCCTransferFrom,
		CancelCCTransferFrom,
		DeleteCCTransferFrom:
		cfg, err := config.FromBytes(cfgBytes)
		if err != nil {
			errMsg := "loading base config " + err.Error()
			span.SetStatus(codes.Error, errMsg)
			return shim.Error(errMsg)
		}

		robotSKIBytes, _ := hex.DecodeString(cfg.GetContract().GetRobotSKI())
		err = hlfcreator.ValidateSKI(robotSKIBytes, creatorSKI, hashedCert)
		if err != nil {
			errMsg := "invoke:unauthorized: robotSKI is not equal creatorSKI and hashedCert: " + err.Error()
			span.SetStatus(codes.Error, errMsg)
			return shim.Error(errMsg)
		}

	case ExecuteTasks:
		bytes, err := TasksExecutorHandler(
			traceCtx,
			stub,
			cfgBytes,
			arguments,
			cc,
		)
		if err != nil {
			errMsg := fmt.Sprintf("failed to execute method %s: txID %s: %s", ExecuteTasks, stub.GetTxID(), err)
			logger.Logger().Error(errMsg)
			span.SetStatus(codes.Error, errMsg)
			return shim.Error(errMsg)
		}

		return shim.Success(bytes)
	}

	method, err := cc.Method(functionName)
	if err != nil {
		errMsg := "invoke: finding method: " + err.Error()
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	// handle invoke and query methods executed without batch process
	if method.Type == contract.MethodTypeInvoke || method.Type == contract.MethodTypeQuery {
		span.SetAttributes(telemetry.MethodType(telemetry.MethodNbTx))
		return cc.noBatchHandler(traceCtx, stub, method, arguments, cfgBytes)
	}

	// handle invoke method with batch process
	return cc.BatchHandler(traceCtx, stub, method, arguments)
}

// ValidateTxID validates the transaction ID to ensure it is correctly formatted.
//
// Args:
// stub: The shim.ChaincodeStubInterface to access the transaction ID.
//
// Returns:
// - nil if the transaction ID is valid.
// - An error if the transaction ID is not valid hexadecimal.
func (cc *Chaincode) ValidateTxID(stub shim.ChaincodeStubInterface) error {
	_, err := hex.DecodeString(stub.GetTxID())
	if err != nil {
		return fmt.Errorf("incorrect tx id: %w", err)
	}

	return nil
}

// BatchHandler handles the batching logic for chaincode invocations.
//
// Args:
// stub: The shim.ChaincodeStubInterface containing the context of the call.
// funcName: The name of the chaincode function to be executed.
// fn: A pointer to the chaincode function to be executed.
// args: A slice of arguments to pass to the function.
//
// Returns:
// - A success response if the batching is successful.
// - An error response if there is any failure in authentication, preparation, or saving to batch.
func (cc *Chaincode) BatchHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	method contract.Method,
	args []string,
) peer.Response {
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.BatchHandler")
	defer span.End()

	span.AddEvent("validating sender")
	sender, args, nonce, err := cc.validateAndExtractInvocationContext(stub, method, args)
	if err != nil {
		span.SetStatus(codes.Error, "validating sender failed")
		return shim.Error(err.Error())
	}

	span.AddEvent("validating arguments")
	if err := cc.Router().Check(method.MethodName, cc.PrependSender(method, sender, args)...); err != nil {
		span.SetStatus(codes.Error, "validating arguments failed")
		return shim.Error(err.Error())
	}

	span.SetAttributes(attribute.String("preimage_tx_id", stub.GetTxID()))
	span.AddEvent("save to batch")
	if err = cc.saveToBatch(traceCtx, stub, method, sender, args, nonce); err != nil {
		span.SetStatus(codes.Error, "save to batch failed")
		return shim.Error(err.Error())
	}

	span.SetStatus(codes.Ok, "")
	return shim.Success(nil)
}

// noBatchHandler is called for functions that should be executed immediately without batching.
// It processes the chaincode function invocation that does not require batch processing.
// This method handles authorization, argument preparation and execution of the chaincode function.
//
// If the function is marked as a 'query', it modifies the stub to ensure that no state changes are persisted.
//
// Returns a shim.Success response if the function invocation is successful. Otherwise, it returns a shim.Error response.
func (cc *Chaincode) noBatchHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	method contract.Method,
	args []string,
	cfgBytes []byte,
) peer.Response {
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.NoBatchHandler")
	defer span.End()

	if method.Type == contract.MethodTypeQuery {
		stub = newQueryStub(stub)
	}

	span.AddEvent("validating sender")
	sender, args, _, err := cc.validateAndExtractInvocationContext(stub, method, args)
	if err != nil {
		span.SetStatus(codes.Error, "validating sender failed")
		return shim.Error(err.Error())
	}

	span.AddEvent("validating arguments")
	if err := cc.Router().Check(method.MethodName, cc.PrependSender(method, sender, args)...); err != nil {
		span.SetStatus(codes.Error, "validating arguments failed")
		return shim.Error(err.Error())
	}

	span.AddEvent("calling method")
	resp, err := cc.InvokeContractMethod(traceCtx, stub, method, sender, args, cfgBytes)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return shim.Error(err.Error())
	}

	span.SetStatus(codes.Ok, "")
	return shim.Success(resp)
}

// batchExecuteHandler is responsible for executing a batch of transactions.
// This handler is invoked when the chaincode function named "batchExecute" is called.
//
// It performs authorization checks using the creator's Subject Key Identifier (SKI) and the hashed certificate
// before proceeding to execute the batch.
//
// Returns a shim.Success response if the batch execution is successful. Otherwise, it returns a shim.Error response
// indicating either an incorrect transaction ID or unauthorized access.
func (cc *Chaincode) batchExecuteHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	creatorSKI [32]byte,
	hashedCert [32]byte,
	args []string,
	cfgBytes []byte,
) peer.Response {
	cfg, err := config.FromBytes(cfgBytes)
	if err != nil {
		return peer.Response{}
	}

	robotSKIBytes, _ := hex.DecodeString(cfg.GetContract().GetRobotSKI())

	err = hlfcreator.ValidateSKI(robotSKIBytes, creatorSKI, hashedCert)
	if err != nil {
		return shim.Error("unauthorized: robotSKI is not equal creatorSKI and hashedCert: " + err.Error())
	}

	return cc.batchExecute(traceCtx, stub, args[0], cfgBytes)
}

// Start begins the chaincode execution based on the environment configuration. It decides whether to
// start the chaincode in the default mode or as a server based on the CHAINCODE_EXEC_MODE environment
// variable. In server mode, it requires the CHAINCODE_ID to be set and uses CHAINCODE_SERVER_PORT for
// the port or defaults to a predefined port if not set. It returns an error if the necessary
// environment variables are not set or if the chaincode fails to start.
func (cc *Chaincode) Start() error {
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
		return errors.New("need to specify chaincode id if running as server")
	}

	port := os.Getenv(chaincodeServerPortEnv)
	if port == "" {
		port = chaincodeServerDefaultPort
	}

	srv := shim.ChaincodeServer{
		CCID:     ccID,
		Address:  fmt.Sprintf("%s:%s", "0.0.0.0", port),
		CC:       cc,
		TLSProps: cc.tls,
	}
	return srv.Start()
}

// readTLSConfigFromEnv tries to read TLS configuration from environment variables.
func readTLSConfigFromEnv() ([]byte, []byte, []byte, error) {
	var (
		key, cert, clientCACerts []byte
		err                      error
	)

	if keyEnv := os.Getenv(tlsKeyEnv); keyEnv != "" {
		key = []byte(keyEnv)
	} else if keyFile := os.Getenv(tlsKeyFileEnv); keyFile != "" {
		key, err = os.ReadFile(keyFile)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read TLS key file: %w", err)
		}
	}

	if certEnv := os.Getenv(tlsCertEnv); certEnv != "" {
		cert = []byte(certEnv)
	} else if certFile := os.Getenv(tlsCertFileEnv); certFile != "" {
		cert, err = os.ReadFile(certFile)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read TLS certificate file: %w", err)
		}
	}

	if caCertsEnv := os.Getenv(tlsClientCACertsEnv); caCertsEnv != "" {
		clientCACerts = []byte(caCertsEnv)
	} else if caCertsFile := os.Getenv(tlsClientCACertsFileEnv); caCertsFile != "" {
		clientCACerts, err = os.ReadFile(caCertsFile)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read client CA certificates file: %w", err)
		}
	}

	return key, cert, clientCACerts, nil
}

func (cc *Chaincode) createIndexHandler(traceCtx telemetry.TraceContext, stub shim.ChaincodeStubInterface, arguments []string) peer.Response {
	_, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.CreateIndexHandler")
	defer span.End()

	if len(arguments) != 1 {
		errMsg := fmt.Sprintf("invoke: incorrect number of arguments: %d", len(arguments))
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	balanceType, err := balance.StringToBalanceType(arguments[0])
	if err != nil {
		errMsg := "invoke: parsing object type: " + err.Error()
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	if err = balance.CreateIndex(stub, balanceType); err != nil {
		errMsg := "invoke: create index: " + err.Error()
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	span.SetStatus(codes.Ok, "")
	return shim.Success([]byte(`{"status": "success"}`))
}

func (cc *Chaincode) PrependSender(method contract.Method, sender *proto.Address, args []string) []string {
	if method.RequiresAuth {
		args = append([]string{sender.AddrString()}, args...)
	}

	return args
}
