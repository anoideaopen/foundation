package core

import (
	"embed"

	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/routing"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// BaseContractInterface represents BaseContract interface
type BaseContractInterface interface { //nolint:interfacebloat
	config.Configurator

	setSrcFs(*embed.FS)

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

	setIsService()
	IsService() bool

	setTracingHandler(th *telemetry.TracingHandler)
	TracingHandler() *telemetry.TracingHandler

	setRouter(routing.Router)
	Router() routing.Router

	GetTraceContext() telemetry.TraceContext
	GetStub() shim.ChaincodeStubInterface

	setEnv(env *environment)
	delEnv()
}
