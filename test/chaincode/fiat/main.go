package main

import (
	"embed"
	"log"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/routing/grpc"
	"github.com/anoideaopen/foundation/core/routing/reflect"
	"github.com/anoideaopen/foundation/test/chaincode/fiat/service"
)

//go:embed *.go
var f embed.FS

func main() {
	l := logger.Logger()
	l.Warning("start fiat")

	var (
		// Creating an instance of the FiatToken contract
		token = NewFiatToken()

		// Initializing the gRPC router. This router will be used to handle
		// method calls defined in gRPC services.
		grpcRouter = grpc.NewRouter()

		// Initializing the reflect router. This router allows dynamic method
		// invocation based on method signatures and names. It is used for
		// methods that are not associated with gRPC.
		reflectRouter = reflect.MustNewRouter(token)
	)

	// Registering the FiatService in the gRPC router.
	// This binds the gRPC methods to their implementation in the FiatToken struct.
	service.RegisterFiatServiceServer(grpcRouter, token)

	// Creating an instance of Chaincode using multiple routers.
	// Here, the core.WithRouters method is used to combine multiple routers,
	// such as grpcRouter and reflectRouter, into a single Chaincode instance.
	cc, err := core.NewCC(
		token,
		core.WithSrcFS(&f),
		core.WithRouters(grpcRouter, reflectRouter),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Starting the Chaincode.
	if err = cc.Start(); err != nil {
		log.Fatal(err)
	}
}
