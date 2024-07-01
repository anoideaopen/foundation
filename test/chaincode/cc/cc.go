package main

import (
	"embed"
	"log"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/token"
)

//go:embed *.go
var f embed.FS

type CcToken struct {
	token.BaseToken
}

func NewCcToken() *CcToken {
	return &CcToken{token.BaseToken{}}
}

func main() {
	l := logger.Logger()
	l.Warning("start cc")

	cc, err := core.NewCC(NewCcToken(), core.WithSrcFS(&f))
	if err != nil {
		log.Fatal(err)
	}

	if err = cc.Start(); err != nil {
		log.Fatal(err)
	}
}
