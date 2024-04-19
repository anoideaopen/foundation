package core

import (
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/hlfcreator"
	"github.com/anoideaopen/foundation/internal/config"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	noBatchPrefix = "NBTx"
	queryPrefix   = "Query"
	txPrefix      = "Tx"
)

const (
	CreateIndex          = "createIndex"
	SetIndexCreatedFlag  = "setIndexCreatedFlag"
	BatchExecute         = "batchExecute"
	SwapDone             = "swapDone"
	MultiSwapDone        = "multiSwapDone"
	CreateCCTransferTo   = "createCCTransferTo"
	DeleteCCTransferTo   = "deleteCCTransferTo"
	CommitCCTransferFrom = "commitCCTransferFrom"
	CancelCCTransferFrom = "cancelCCTransferFrom"
	DeleteCCTransferFrom = "deleteCCTransferFrom"
)

// TokenConfigurable is an interface that defines methods for validating, applying, and
// retrieving token configuration.
type TokenConfigurable interface {
	// ValidateTokenConfig validates the provided token configuration data.
	// It takes a byte slice containing JSON-encoded token configuration and returns an error
	// if the validation fails. The error should provide information about the validation failure.
	//
	// The implementation of this method may include unmarshalling the JSON-encoded data and
	// invoking specific validation logic on the deserialized token configuration.
	ValidateTokenConfig(config []byte) error

	// ApplyTokenConfig applies the provided token configuration to the implementing object.
	// It takes a pointer to a proto.TokenConfig struct and returns an error if applying the
	// configuration fails. The error should provide information about the failure.
	//
	// The implementation of this method may include setting various properties of the implementing
	// object based on the values in the provided token configuration.
	ApplyTokenConfig(config *proto.TokenConfig) error

	// TokenConfig retrieves the current token configuration from the implementing object.
	// It returns a pointer to a proto.TokenConfig struct representing the current configuration.
	//
	// The implementation of this method should return the current state of the token configuration
	// stored within the object.
	TokenConfig() *proto.TokenConfig
}

// ExternalConfigurable is an interface that defines methods for validating and applying external configuration.
// This interface should be implemented in chaincode to initialize some extended chaincode attributes
// on Init() call.
// ValidateExtConfig function called in shim.Chaincode.Init function, verify that config is OK.
// ApplyExtConfig function called each time shim.Chaincode.Invoke called. It loads configuration from state
// and apply to chaincode.
type ExternalConfigurable interface {
	// ValidateExtConfig validates the provided external configuration data.
	// It takes a byte slice containing the external configuration data, typically in a
	// specific format, and returns an error if the validation fails. The error should
	// provide information about the validation failure.
	ValidateExtConfig(cfgBytes []byte) error

	// ApplyExtConfig applies the provided external configuration to the chaincode.
	// It takes a byte slice containing the external configuration data and returns an error
	// if applying the configuration fails. The error should provide information about the failure.
	ApplyExtConfig(cfgBytes []byte) error
}

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
	SrcFs *embed.FS // SrcFs is a file system that contains the source files for the chaincode.
	TLS   *TLS      // TLS contains the TLS configuration for the chaincode.
}

// ChainCode defines the structure for a chaincode instance, with methods,
// configuration, and options for transaction processing.
type ChainCode struct {
	contract BaseContractInterface // Contract interface containing the chaincode logic.
	tls      shim.TLSProperties    // TLS configuration properties.
	// methods stores contract public methods, filled through parseContractMethods.
	methods ContractMethods
}

// WithSrcFS is a ChaincodeOption that specifies the source file system to be used by the ChainCode.
//
// fs: A pointer to an embedded file system containing the chaincode files.
//
// It returns a ChaincodeOption that sets the SrcFs field in the chaincodeOptions.
func WithSrcFS(fs *embed.FS) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.SrcFs = fs
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
) (*ChainCode, error) {
	empty := new(ChainCode) // Empty chaincode result fixes integration tests.

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
	cc.setSrcFs(chOpts.SrcFs)

	// Set up the ChainCode structure.
	out := &ChainCode{
		contract: cc,
		tls:      tlsProps,
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
func (cc *ChainCode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	creator, err := stub.GetCreator()
	if err != nil {
		return shim.Error("init: getting creator of transaction: " + err.Error())
	}
	if err = hlfcreator.ValidateAdminCreator(creator); err != nil {
		return shim.Error("init: validating admin creator: " + err.Error())
	}

	args := stub.GetStringArgs()

	var cfgBytes []byte
	if config.IsJSONConfig(args) {
		cfgBytes = []byte(args[0])
	} else {
		// handle args as position parameters and fill config structure.
		// TODO: remove this code when all users moved to json-config initialization.
		cfgBytes, err = config.ParseArgsArr(stub.GetChannelID(), args)
		if err != nil {
			return shim.Error(fmt.Sprintf("init: parsing args old way: %s", err))
		}
	}

	if err = validateContractMethods(cc.contract); err != nil {
		return shim.Error("init: validating contract methods: " + err.Error())
	}

	if c, ok := cc.contract.(ContractConfigurable); ok {
		if err = c.ValidateConfig(cfgBytes); err != nil {
			return shim.Error(fmt.Sprintf("init: validating base config: %s", err))
		}
	} else {
		return shim.Error("chaincode does not implement ContractConfigurable interface")
	}

	if t, ok := cc.contract.(TokenConfigurable); ok {
		if err = t.ValidateTokenConfig(cfgBytes); err != nil {
			return shim.Error(fmt.Sprintf("init: validating token config: %s", err))
		}
	}

	if tc, ok := cc.contract.(ExternalConfigurable); ok {
		if err = tc.ValidateExtConfig(cfgBytes); err != nil {
			return shim.Error(fmt.Sprintf("init: validating extended token config: %s", err))
		}
	}

	if err = config.SaveConfig(stub, cfgBytes); err != nil {
		return shim.Error("init: saving config: " + err.Error())
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
func (cc *ChainCode) Invoke(stub shim.ChaincodeStubInterface) (r peer.Response) {
	r = shim.Error("panic invoke")
	defer func() {
		if rc := recover(); rc != nil {
			log.Printf("panic invoke\nrc: %v\nstack: %s\n", rc, debug.Stack())
		}
	}()

	start := time.Now()

	// getting contract config
	cfgBytes, err := config.LoadRawConfig(stub)
	if err != nil {
		return shim.Error("invoke: loading raw config: " + err.Error())
	}

	// Apply config on all layers: base contract (SKI's & chaincode options),
	// token base attributes and extended token parameters.
	if err = applyConfig(&cc.contract, stub, cfgBytes); err != nil {
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

	span.AddEvent("parsing contract methods")
	cc.methods, err = parseContractMethods(cc.contract)
	if err != nil {
		errMsg := "invoke: parsing contract methods: " + err.Error()
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	// it is probably worth checking if the function is not locked before it is executed.
	// You should also check with swap and multiswap locking and
	// display the error explicitly instead of saying that the function was not found.
	span.SetAttributes(attribute.String("method", functionName))
	switch functionName {
	case CreateIndex: // Creating a reverse index to find token owners.
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

	case BatchExecute:
		return cc.batchExecuteHandler(traceCtx, stub, creatorSKI, hashedCert, arguments, cfgBytes)

	case SwapDone:
		return cc.swapDoneHandler(traceCtx, stub, arguments, cfgBytes)

	case MultiSwapDone:
		return cc.multiSwapDoneHandler(traceCtx, stub, arguments, cfgBytes)

	case CreateCCTransferTo,
		DeleteCCTransferTo,
		CommitCCTransferFrom,
		CancelCCTransferFrom,
		DeleteCCTransferFrom:
		contractCfg, err := config.ContractConfigFromBytes(cfgBytes)
		if err != nil {
			errMsg := "loading base config " + err.Error()
			span.SetStatus(codes.Error, errMsg)
			return shim.Error(errMsg)
		}

		robotSKIBytes, _ := hex.DecodeString(contractCfg.GetRobotSKI())
		err = hlfcreator.ValidateSKI(robotSKIBytes, creatorSKI, hashedCert)
		if err != nil {
			errMsg := "invoke:unauthorized: robotSKI is not equal creatorSKI and hashedCert: " + err.Error()
			span.SetStatus(codes.Error, errMsg)
			return shim.Error(errMsg)
		}
	}

	method, err := cc.methods.Method(functionName)
	if err != nil {
		errMsg := "invoke: finding method: " + err.Error()
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	// handle invoke and query methods executed without batch process
	if method.noBatch {
		span.SetAttributes(telemetry.MethodType(telemetry.MethodNbTx))
		return cc.noBatchHandler(traceCtx, stub, functionName, method, arguments, cfgBytes)
	}

	// handle invoke method with batch process
	return cc.BatchHandler(traceCtx, stub, functionName, method, arguments)
}

// ValidateTxID validates the transaction ID to ensure it is correctly formatted.
//
// Args:
// stub: The shim.ChaincodeStubInterface to access the transaction ID.
//
// Returns:
// - nil if the transaction ID is valid.
// - An error if the transaction ID is not valid hexadecimal.
func (cc *ChainCode) ValidateTxID(stub shim.ChaincodeStubInterface) error {
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
func (cc *ChainCode) BatchHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	funcName string,
	fn *Fn,
	args []string,
) peer.Response {
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.BatchHandler")
	defer span.End()

	span.AddEvent("validating sender")
	sender, args, nonce, err := cc.validateAndExtractInvocationContext(stub, fn, funcName, args)
	if err != nil {
		span.SetStatus(codes.Error, "validating sender failed")
		return shim.Error(err.Error())
	}
	span.AddEvent("prepare to save")
	args, err = doPrepareToSave(stub, fn, args)
	if err != nil {
		span.SetStatus(codes.Error, "prepare to save failed")
		return shim.Error(err.Error())
	}

	span.SetAttributes(attribute.String("preimage_tx_id", stub.GetTxID()))
	span.AddEvent("save to batch")
	if err = cc.saveToBatch(traceCtx, stub, funcName, fn, sender, args[:len(fn.in)], nonce); err != nil {
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
func (cc *ChainCode) noBatchHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	funcName string,
	fn *Fn,
	args []string,
	cfgBytes []byte,
) peer.Response {
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.NoBatchHandler")
	defer span.End()

	if fn.query {
		stub = newQueryStub(stub)
	}

	span.AddEvent("validating sender")
	sender, args, _, err := cc.validateAndExtractInvocationContext(stub, fn, funcName, args)
	if err != nil {
		span.SetStatus(codes.Error, "validating sender failed")
		return shim.Error(err.Error())
	}
	span.AddEvent("prepare to save")
	args, err = doPrepareToSave(stub, fn, args)
	if err != nil {
		span.SetStatus(codes.Error, "prepare to save failed")
		return shim.Error(err.Error())
	}

	span.AddEvent("calling method")
	resp, err := cc.callMethod(traceCtx, stub, fn, sender, args, cfgBytes)
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
func (cc *ChainCode) batchExecuteHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	creatorSKI [32]byte,
	hashedCert [32]byte,
	args []string,
	cfgBytes []byte,
) peer.Response {
	contractCfg, err := config.ContractConfigFromBytes(cfgBytes)
	if err != nil {
		return peer.Response{}
	}

	robotSKIBytes, _ := hex.DecodeString(contractCfg.GetRobotSKI())

	err = hlfcreator.ValidateSKI(robotSKIBytes, creatorSKI, hashedCert)
	if err != nil {
		return shim.Error("unauthorized: robotSKI is not equal creatorSKI and hashedCert: " + err.Error())
	}

	return cc.batchExecute(traceCtx, stub, args[0], cfgBytes)
}

// callMethod invokes a method on the ChainCode contract using reflection. It converts the
// arguments from strings to their expected types, handles sender address if provided,
// creates a copy of the contract with provided initialization arguments, and then
// calls the specified method on the contract.
//
// Returns:
//   - A byte slice containing the JSON-marshaled return value of the method, if method.out is true.
//   - An error if there is any issue with argument conversion, contract copying, or method call.
//
// If 'sender' is non-nil, it is converted to a types.Sender and prepended to the argument list.
// If 'method.out' is true, the return value of the method call is JSON-marshaled and returned as a byte slice.
// If the method call returns an error or the return value cannot be converted to an error, it is returned as is.
//
// Errors from the method call are converted to Go errors and returned. If the conversion is not possible,
// a generic error with message requireInterfaceErrMsg is returned.
func (cc *ChainCode) callMethod(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	method *Fn,
	sender *proto.Address,
	args []string,
	cfgBytes []byte,
) ([]byte, error) {
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.CallMethod")
	defer span.End()

	span.SetAttributes(attribute.StringSlice("args", args))
	span.AddEvent("convert to call")
	values, err := doConvertToCall(stub, method, args)
	if err != nil {
		return nil, err
	}
	if sender != nil {
		span.SetAttributes(attribute.String("sender addr", sender.AddrString()))
		values = append([]reflect.Value{
			reflect.ValueOf(types.NewSenderFromAddr((*types.Address)(sender))),
		}, values...)
	}

	span.AddEvent("copy contract")
	contract, _ := copyContractWithConfig(traceCtx, cc.contract, stub, cfgBytes)

	span.AddEvent("call")
	out := method.fn.Call(append([]reflect.Value{contract}, values...))
	errInt := out[0].Interface()
	if method.out {
		errInt = out[1].Interface()
	}
	if errInt != nil {
		err, ok := errInt.(error)
		if !ok {
			span.SetStatus(codes.Error, requireInterfaceErrMsg)
			return nil, errors.New(requireInterfaceErrMsg)
		}
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if method.out {
		span.SetStatus(codes.Ok, "")
		return json.Marshal(out[0].Interface())
	}
	span.SetStatus(codes.Ok, "")
	return nil, nil
}

// doConvertToCall prepares the arguments for a chaincode method call by converting each
// string argument to its expected Go type as defined in the method's 'in' field. It uses reflection
// to dynamically convert arguments and checks if the number of provided arguments matches the expectation.
//
// It returns a slice of reflect.Value representing the converted arguments, and an error if
// the conversion fails or if there is an incorrect number of arguments.
func doConvertToCall(
	stub shim.ChaincodeStubInterface,
	method *Fn,
	args []string,
) ([]reflect.Value, error) {
	found := len(args)
	expected := len(method.in)
	if found < expected {
		return nil, fmt.Errorf(
			"incorrect number of arguments, found %d but expected more than %d",
			found,
			expected,
		)
	}
	// todo check is args enough
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
				return nil, errors.New(requireInterfaceErrMsg)
			}
			return nil, fmt.Errorf(
				"failed to convert arg value '%s' to type '%s' on index '%d': %w",
				args[i], impl.String(), i, err,
			)
		}
		vArgs[i] = res[0]
	}
	return vArgs, nil
}

// doPrepareToSave prepares the arguments for storage by ensuring they are in the correct format.
// It checks if there are enough arguments and uses either a custom 'prepareToSave' or 'convertToCall'
// method defined in the method's 'in' field for conversion. It returns the processed arguments as a
// slice of strings, and an error if the conversion fails or if there is an incorrect number of arguments.
func doPrepareToSave(
	stub shim.ChaincodeStubInterface,
	method *Fn,
	args []string,
) ([]string, error) {
	if len(args) < len(method.in) {
		return nil, fmt.Errorf(
			"incorrect number of arguments. current count of args is %d but expected more than %d",
			len(args),
			len(method.in),
		)
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
					return nil, errors.New(requireInterfaceErrMsg)
				}
				return nil, err
			}
			as[i], ok = res[0].Interface().(string)
			if !ok {
				return nil, errors.New(requireInterfaceErrMsg)
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
				return nil, errors.New(requireInterfaceErrMsg)
			}
			return nil, err
		}

		as[i] = args[i] // in this case we don't convert argument
	}
	return as, nil
}

// copyContract creates a deep copy of a contract's interface. It uses reflection to copy each field
// from the original to a new instance of the same type. The new copy is initialized with the provided
// stub, cfgBytes and noncePrefix. It returns a reflect.Value of the copied contract and
// an interface to the copied contract.
func copyContractWithConfig(
	traceCtx telemetry.TraceContext,
	orig BaseContractInterface,
	stub shim.ChaincodeStubInterface,
	cfgBytes []byte,
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

	_ = applyConfig(&contract, stub, cfgBytes)

	contract.setTraceContext(traceCtx)
	contract.setTracingHandler(contract.TracingHandler())

	return cp, contract
}

// Start begins the chaincode execution based on the environment configuration. It decides whether to
// start the chaincode in the default mode or as a server based on the CHAINCODE_EXEC_MODE environment
// variable. In server mode, it requires the CHAINCODE_ID to be set and uses CHAINCODE_SERVER_PORT for
// the port or defaults to a predefined port if not set. It returns an error if the necessary
// environment variables are not set or if the chaincode fails to start.
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

// applyConfig applies the provided configuration to a BaseContractInterface, updating its internal state.
//
// The function takes a BaseContractInterface (bci), a shim.ChaincodeStubInterface (stub), and the
// configuration data in the form of a byte slice (cfgBytes). It sets the ChaincodeStubInterface on
// the BaseContractInterface and tries to apply Base, Token and Extended configurations
// based on the implemented interfaces of the
// ContractConfigurable, TokenConfigurable and ExternalConfigurable consequently.
//
// If the BaseContractInterface does not implement the ContractConfigurable interface, an error is returned.
//
// If any step in the configuration application process fails, an error is returned with details about
// the specific error encountered.
func applyConfig(
	bci *BaseContractInterface,
	stub shim.ChaincodeStubInterface,
	cfgBytes []byte,
) error {
	// WARN: if stub is not set,
	// it should be thrown through it into all methods before CallMethod.
	(*bci).setStub(stub)

	ccbc, ok := (*bci).(ContractConfigurable)
	if !ok {
		return errors.New("chaincode is not ContractConfigurable")
	}

	contractCfg, err := config.ContractConfigFromBytes(cfgBytes)
	if err != nil {
		return fmt.Errorf("parsing base config: %w", err)
	}

	if contractCfg.GetOptions() == nil {
		contractCfg.Options = new(proto.ChaincodeOptions)
	}

	if err = ccbc.ApplyContractConfig(contractCfg); err != nil {
		return fmt.Errorf("applying base config: %w", err)
	}

	if tc, ok := (*bci).(TokenConfigurable); ok {
		tokenCfg, err := config.TokenConfigFromBytes(cfgBytes)
		if err != nil {
			return fmt.Errorf("parsing token config: %w", err)
		}

		if err = tc.ApplyTokenConfig(tokenCfg); err != nil {
			return fmt.Errorf("applying token config: %w", err)
		}
	}

	if ec, ok := (*bci).(ExternalConfigurable); ok {
		if err = ec.ApplyExtConfig(cfgBytes); err != nil {
			return err
		}
	}

	return nil
}
