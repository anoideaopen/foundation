package unit

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock/stub"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/anoideaopen/foundation/token"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestDuplicateNames(t *testing.T) {
	chName := "DN"

	cfg := &pb.Config{
		Contract: &pb.ContractConfig{
			Symbol:   chName,
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    fixtures_test.Admin,
		},
		Token: &pb.TokenConfig{
			Name:     chName + " Token",
			Decimals: 8,
			Issuer:   fixtures_test.Admin,
		},
	}

	cfgBytes, _ := protojson.Marshal(cfg)

	tt := []struct {
		name string
		bci  core.BaseContractInterface
		err  error
	}{
		{
			name: "no duplicated functions",
			bci:  &token.BaseToken{},
			err:  nil,
		},
		{
			name: "variant #1",
			bci:  &DuplicateNamesT1{},
			err:  fmt.Errorf("%w, method: '%s'", core.ErrMethodAlreadyDefined, "allowedBalanceAdd"),
		},
		{
			name: "variant #2",
			bci:  &DuplicateNamesT2{},
			err:  fmt.Errorf("%w, method: '%s'", core.ErrMethodAlreadyDefined, "allowedBalanceAdd"),
		},
		{
			name: "variant #3",
			bci:  &DuplicateNamesT3{},
			err:  fmt.Errorf("%w, method: '%s'", core.ErrMethodAlreadyDefined, "allowedBalanceAdd"),
		},
	}

	t.Parallel()
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			cc, _ := core.NewCC(test.bci)
			ms := stub.NewMockStub(chName, cc)

			_ = ms.SetAdminCreatorCert("platformMSP")

			idBytes := [16]byte(uuid.New())
			rsp := ms.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{cfgBytes})
			if test.err == nil {
				assert.Empty(t, rsp.GetMessage())
			} else {
				assert.Equal(t, int32(shim.ERROR), rsp.GetStatus())
				assert.Contains(t, rsp.GetMessage(), test.err.Error())
			}
		})
	}
}

// Tokens with some duplicate names in methods

type DuplicateNamesT1 struct {
	token.BaseToken
}

func (dnt *DuplicateNamesT1) NBTxAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return dnt.AllowedBalanceAdd(token, address, amount, reason)
}

func (dnt *DuplicateNamesT1) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) (string, error) {
	return "Ok", dnt.AllowedBalanceAdd(token, address, amount, reason)
}

type DuplicateNamesT2 struct {
	token.BaseToken
}

func (dnt *DuplicateNamesT2) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) (string, error) {
	return "Ok", dnt.AllowedBalanceAdd(token, address, amount, reason)
}

func (dnt *DuplicateNamesT2) TxAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return dnt.AllowedBalanceAdd(token, address, amount, reason)
}

type DuplicateNamesT3 struct {
	token.BaseToken
}

func (dnt *DuplicateNamesT3) NBTxAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return dnt.AllowedBalanceAdd(token, address, amount, reason)
}

func (dnt *DuplicateNamesT3) TxAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return dnt.AllowedBalanceAdd(token, address, amount, reason)
}
