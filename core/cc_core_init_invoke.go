package core

import (
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/hlfcreator"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

// Init is called during chaincode instantiation to initialize any data. Note that upgrade
// also calls this function to reset or to migrate data.
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

	if err = config.Validate(cc.contract, cfgBytes); err != nil {
		return shim.Error("init: validating config: " + err.Error())
	}

	if err = config.Save(stub, cfgBytes); err != nil {
		return shim.Error("init: saving config: " + err.Error())
	}

	return shim.Success(nil)
}

// Invoke is called to update or query the ledger in a proposal transaction. Given the
// function name, it delegates the execution to the respective handler.
func (cc *Chaincode) Invoke(stub shim.ChaincodeStubInterface) (r peer.Response) {
	r = shim.Error("panic invoke")
	log := logger.Logger()
	defer func() {
		if rc := recover(); rc != nil {
			log.Errorf("panic invoke\nrc: %v\nstack: %s\n", rc, debug.Stack())
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
	if err = config.Configure(cc.contract, cfgBytes); err != nil {
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

	span.AddEvent("get function")
	function, _ := stub.GetFunctionAndParameters()

	span.AddEvent(fmt.Sprintf("begin id: %s, name: %s", transactionID, function))
	defer func() {
		span.AddEvent(fmt.Sprintf("end id: %s, name: %s, elapsed: %d",
			transactionID,
			function,
			time.Since(start),
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
	span.SetAttributes(attribute.String("method", function))
	switch function {
	case CreateIndex: // Creating a reverse index to find token owners.
		return cc.createIndexHandler(traceCtx, stub)

	case BatchExecute:
		defer func() {
			log.Warningf("tx id: %s, name: %s, elapsed: %s",
				transactionID,
				function,
				time.Since(start),
			)
		}()
		return cc.batchExecuteHandler(traceCtx, stub, creatorSKI, hashedCert)

	case SwapDone:
		cc.contract.setEnv(&environment{
			stub:  stub,
			trace: traceCtx,
		})
		defer cc.contract.delEnv()

		return cc.swapDoneHandler(stub)

	case MultiSwapDone:
		cc.contract.setEnv(&environment{
			stub:  stub,
			trace: traceCtx,
		})
		defer cc.contract.delEnv()

		return cc.multiSwapDoneHandler(stub)

	case
		CreateCCTransferTo,
		DeleteCCTransferTo,
		CommitCCTransferFrom,
		CancelCCTransferFrom,
		DeleteCCTransferFrom:
		robotSKIBytes, _ := hex.DecodeString(cc.contract.ContractConfig().GetRobotSKI())

		err = hlfcreator.ValidateSKI(robotSKIBytes, creatorSKI, hashedCert)
		if err != nil {
			errMsg := "invoke: unauthorized: robotSKI is not equal creatorSKI and hashedCert: " + err.Error()
			span.SetStatus(codes.Error, errMsg)
			return shim.Error(errMsg)
		}

	case ExecuteTasks:
		defer func() {
			log.Warningf("tx id: %s, name: %s, elapsed: %s",
				transactionID,
				function,
				time.Since(start),
			)
		}()
		bytes, err := TasksExecutorHandler(
			traceCtx,
			stub,
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

	method := cc.Router().Method(function)
	if method == "" {
		errMsg := fmt.Sprintf("invoke: finding method: method '%s' not found", function)
		span.SetStatus(codes.Error, errMsg)
		return shim.Error(errMsg)
	}

	if cc.contract.ContractConfig().GetOptions() != nil {
		var (
			swapMethods      = []string{"QuerySwapGet", "TxSwapBegin", "TxSwapCancel"}
			multiSwapMethods = []string{"QueryMultiSwapGet", "TxMultiSwapBegin", "TxMultiSwapCancel"}
			opts             = cc.contract.ContractConfig().GetOptions()
		)

		if OneOf(method, opts.GetDisabledFunctions()...) ||
			(opts.GetDisableSwaps() && OneOf(method, swapMethods...)) ||
			(opts.GetDisableMultiSwaps() && OneOf(method, multiSwapMethods...)) {
			return shim.Error(fmt.Sprintf("invoke: finding method: method '%s' not found", function))
		}
	}

	// handle invoke and query methods executed without batch process
	if cc.Router().IsInvoke(method) || cc.Router().IsQuery(method) {
		span.SetAttributes(telemetry.MethodType(telemetry.MethodNbTx))
		return cc.noBatchHandler(traceCtx, stub)
	}

	// handle invoke method with batch process
	return cc.BatchHandler(traceCtx, stub)
}
