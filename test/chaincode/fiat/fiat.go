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
		token         = NewFiatToken()
		grpcRouter    = grpc.NewRouter()
		reflectRouter = reflect.MustNewRouter(token)
	)

	service.RegisterFiatServiceServer(grpcRouter, token)

	cc, err := core.NewCC(
		token,
		core.WithSrcFS(&f),
		core.WithRouters(grpcRouter, reflectRouter),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cc.Start(); err != nil {
		log.Fatal(err)
	}
}
