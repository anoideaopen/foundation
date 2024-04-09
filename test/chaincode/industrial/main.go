package main

import (
	"embed"
	"errors"
	"log"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	industrialtoken "github.com/anoideaopen/foundation/test/chaincode/industrial/industrial_token"
)

//go:embed *.go industrial_token/*.go
var f embed.FS

// IT - industrial token base struct
type IT struct {
	industrialtoken.IndustrialToken
}

func NewIT() *IT {
	return &IT{industrialtoken.IndustrialToken{}}
}

var groups = []industrialtoken.Group{
	{
		ID:       "202009",
		Emission: 10000000000000,
		Maturity: "21.09.2020 22:00:00",
		Note:     "Test note",
	}, {
		ID:       "202010",
		Emission: 100000000000000,
		Maturity: "21.10.2020 22:00:00",
		Note:     "Test note",
	}, {
		ID:       "202011",
		Emission: 200000000000000,
		Maturity: "21.11.2020 22:00:00",
		Note:     "Test note",
	}, {
		ID:       "202012",
		Emission: 50000000000000,
		Maturity: "21.12.2020 22:00:00",
		Note:     "Test note",
	},
}

// NBTxInitialize - initializes chaincode
func (it *IT) NBTxInitialize(sender *types.Sender) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}

	return it.Initialize(groups)
}

func main() {
	cc, err := core.NewCC(NewIT(), core.WithSrcFS(&f))
	if err != nil {
		log.Fatal(err)
	}

	if err = cc.Start(); err != nil {
		log.Fatal(err)
	}
}
