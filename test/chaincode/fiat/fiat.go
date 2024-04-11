package main

import (
	"embed"
	"log"

	"github.com/anoideaopen/foundation/core"
)

//go:embed *.go
var f embed.FS

func main() {
	cc, err := core.NewCC(NewFiatToken(), core.WithSrcFS(&f))
	if err != nil {
		log.Fatal(err)
	}

	if err = cc.Start(); err != nil {
		log.Fatal(err)
	}
}
