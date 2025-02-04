package core

import (
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/routing"
	"github.com/anoideaopen/foundation/core/routing/mux"
	"github.com/anoideaopen/foundation/core/routing/reflect"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/hlfcreator"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
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

// Chaincode defines the structure for a chaincode instance, with methods,
// configuration, and options for transaction processing.
type Chaincode struct {
	contract     BaseContractInterface // Contract interface containing the chaincode logic.
	configMapper config.ConfigMapper   // ConfigMapper maps the arguments to a proto.Config instance.
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

	// Apply chaincode options provided by the caller.
	chOpts := chaincodeOptions{}
	for _, option := range chOptions {
		if option == nil {
			continue
		}
		err := option(&chOpts)
		if err != nil {
			return empty, fmt.Errorf("reading opts: %w", err)
		}
	}

	// Initialize the contract.
	cc.setSrcFs(chOpts.SrcFS)

	// Set up the router.
	var (
		router routing.Router
		err    error
	)
	if len(chOpts.Routers) > 0 {
		router, err = mux.NewRouter(chOpts.Routers...)
	} else {
		router, err = reflect.NewRouter(cc)
	}

	if err != nil {
		return empty, err
	}

	cc.setRouter(router)

	// Set up the ChainCode structure.
	out := &Chaincode{
		contract:     cc,
		configMapper: chOpts.ConfigMapper,
	}

	return out, nil
}

// Router returns the contract router for the Chaincode.
func (cc *Chaincode) Router() routing.Router {
	return cc.contract.Router()
}

// ChaincodeOption represents a function that applies configuration options to
// a chaincodeOptions object.
type ChaincodeOption func(opts *chaincodeOptions) error

// chaincodeOptions is a structure that holds advanced options for configuring
// a ChainCode instance.
type chaincodeOptions struct {
	SrcFS        *embed.FS           // SrcFS is a file system that contains the source files for the chaincode.
	TLS          *TLS                // TLS contains the TLS configuration for the chaincode.
	ConfigMapper config.ConfigMapper // ConfigMapper maps the arguments to a proto.Config instance.
	Routers      []routing.Router    // Routers is a list of routers for the chaincode.
}

// WithRouter returns a ChaincodeOption function that sets the router in the chaincode options.
func WithRouter(router routing.Router) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.Routers = append(o.Routers, router)
		return nil
	}
}

// WithRouters returns a ChaincodeOption function that sets the router in the chaincode options.
func WithRouters(routers ...routing.Router) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.Routers = append(o.Routers, routers...)
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
func WithConfigMapperFunc(cmf config.ConfigMapperFunc) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.ConfigMapper = cmf
		return nil
	}
}

// WithConfigMapper is a ChaincodeOption that specifies the ConfigMapper for the ChainCode.
func WithConfigMapper(cm config.ConfigMapper) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.ConfigMapper = cm
		return nil
	}
}

// WithSrcFS is a ChaincodeOption that specifies the source file system to be used by the ChainCode.
func WithSrcFS(fs *embed.FS) ChaincodeOption {
	return func(o *chaincodeOptions) error {
		o.SrcFS = fs
		return nil
	}
}

// TLS holds the key and certificate data for TLS communication, as well as
// client CA certificates for peer verification if needed.
type TLS struct {
	Key           []byte // Private key for TLS authentication.
	Cert          []byte // Public certificate for TLS authentication.
	ClientCACerts []byte // Optional client CA certificates for verifying connecting peers.
}

// WithTLS is a ChaincodeOption that specifies the TLS configuration for the ChainCode.
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
func (cc *Chaincode) BatchHandler(traceCtx telemetry.TraceContext, stub shim.ChaincodeStubInterface) *peer.Response {
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.BatchHandler")
	defer span.End()

	fn, args := stub.GetFunctionAndParameters()

	span.AddEvent("validating sender")
	sender, invocationArgs, nonce, err := cc.validateAndExtractInvocationContext(stub, fn, args)
	if err != nil {
		span.SetStatus(codes.Error, "validating sender failed")
		return shim.Error(err.Error())
	}

	method := cc.Router().Method(fn)

	span.AddEvent("validating arguments")
	if err = cc.Router().Check(stub, method, cc.PrependSender(method, sender, invocationArgs)...); err != nil {
		span.SetStatus(codes.Error, "validating arguments failed")
		return shim.Error(err.Error())
	}

	span.SetAttributes(attribute.String("preimage_tx_id", stub.GetTxID()))
	span.AddEvent("save to batch")
	if err = cc.saveToBatch(traceCtx, stub, fn, sender, invocationArgs, nonce); err != nil {
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
) *peer.Response {
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.NoBatchHandler")
	defer span.End()

	fn, args := stub.GetFunctionAndParameters()

	span.AddEvent("validating sender")
	sender, invocationArgs, _, err := cc.validateAndExtractInvocationContext(stub, fn, args)
	if err != nil {
		span.SetStatus(codes.Error, "validating sender failed")
		return shim.Error(err.Error())
	}

	method := cc.Router().Method(fn)
	if cc.Router().IsQuery(method) {
		stub = newQueryStub(stub)
	}

	span.AddEvent("validating arguments")
	if err = cc.Router().Check(stub, method, cc.PrependSender(method, sender, invocationArgs)...); err != nil {
		span.SetStatus(codes.Error, "validating arguments failed")
		return shim.Error(err.Error())
	}

	span.AddEvent("calling method")
	resp, err := cc.InvokeContractMethod(traceCtx, stub, sender, method, invocationArgs)
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
) *peer.Response {
	robotSKIBytes, _ := hex.DecodeString(cc.contract.ContractConfig().GetRobotSKI())

	err := hlfcreator.ValidateSKI(robotSKIBytes, creatorSKI, hashedCert)
	if err != nil {
		return shim.Error("unauthorized: robotSKI is not equal creatorSKI and hashedCert: " + err.Error())
	}

	_, args := stub.GetFunctionAndParameters()

	return cc.batchExecute(traceCtx, stub, args[0])
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

	// if exec mode is chaincode-as-service, set the parameter isService in the base contract to true
	cc.contract.setIsService()

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

	tlsProps, err := tlsProperties()
	if err != nil {
		return fmt.Errorf("failed obtaining tls properties for chaincode server: %w", err)
	}

	srv := shim.ChaincodeServer{
		CCID:     ccID,
		Address:  fmt.Sprintf("%s:%s", "0.0.0.0", port),
		CC:       cc,
		TLSProps: tlsProps,
	}
	return srv.Start()
}

func tlsProperties() (shim.TLSProperties, error) {
	tlsProps := shim.TLSProperties{
		Disabled: true,
	}

	key, cert, clientCACerts, err := readTLSConfigFromEnv()
	if err != nil {
		return tlsProps, fmt.Errorf("error reading TLS config from environment: %w", err)
	}

	// If TLS configuration is found in environment variables, use it.
	if key != nil && cert != nil {
		tlsProps.Disabled = false
		tlsProps.Key = key
		tlsProps.Cert = cert
		tlsProps.ClientCACerts = clientCACerts
	}

	return tlsProps, nil
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

func (cc *Chaincode) createIndexHandler(traceCtx telemetry.TraceContext, stub shim.ChaincodeStubInterface) *peer.Response {
	_, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.CreateIndexHandler")
	defer span.End()

	_, args := stub.GetFunctionAndParameters()

	if len(args) != 1 {
		errMsg := fmt.Sprintf("invoke: incorrect number of arguments: %d", len(args))
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	balanceType, err := balance.StringToBalanceType(args[0])
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

func (cc *Chaincode) PrependSender(method string, sender *proto.Address, args []string) []string {
	if cc.Router().AuthRequired(method) {
		args = append([]string{sender.AddrString()}, args...)
	}

	return args
}
