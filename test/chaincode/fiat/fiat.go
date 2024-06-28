package main

import (
	"embed"
	"log"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/grpc"
	"github.com/anoideaopen/foundation/test/chaincode/fiat/service"
)

//go:embed *.go
var f embed.FS

func main() {
	token := NewFiatToken()

	router := grpc.NewRouter(
		grpc.RouterConfig{Fallback: grpc.DefaultReflectxFallback(token)},
	)

	service.RegisterFiatServiceServer(router, token)

	cc, err := core.NewCC(
		token,
		core.WithSrcFS(&f),
		core.WithRouter(router),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cc.Start(); err != nil {
		log.Fatal(err)
	}
}
