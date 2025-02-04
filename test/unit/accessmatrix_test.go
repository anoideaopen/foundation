package unit

import (
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/acl"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type IssuerCheckerToken struct {
	token.BaseToken
}

const (
	fnGetRight                  = "getRight"
	fnGetAddressRightForNominee = "getAddressRightForNominee"
)

func (ict *IssuerCheckerToken) QueryGetRight(ccname string, address *types.Address, role, operation string) (bool, error) {
	stub := ict.GetStub()
	r := acl.Role(role)
	params := []string{stub.GetChannelID(), ccname, r.String(), operation, address.String()}
	right, err := acl.GetAccountRight(ict.GetStub(), params)
	if err != nil {
		return false, err
	}

	if right.HaveRight {
		return true, nil
	}

	return false, err
}

func (ict *IssuerCheckerToken) QueryGetAddressRightForNominee(chaincodeName string, nomineeAddress, principalAddress *types.Address) (bool, error) {
	stub := ict.GetStub()
	params := []string{stub.GetChannelID(), chaincodeName, nomineeAddress.String(), principalAddress.String()}
	right, err := acl.GetAddressRightForNominee(ict.GetStub(), params)
	if err != nil {
		return false, err
	}

	if right.HaveRight {
		return true, nil
	}

	return false, err
}

func (ict *IssuerCheckerToken) QueryGetAddressesListForNominee(chaincodeName string, nomineeAddress *types.Address) ([]*types.Address, error) {
	stub := ict.GetStub()
	params := []string{stub.GetChannelID(), chaincodeName, nomineeAddress.String()}
	accounts, err := acl.GetAddressesListForNominee(ict.GetStub(), params)
	if err != nil {
		return nil, err
	}

	var addresses []*types.Address
	for _, addr := range accounts.GetAddresses() {
		addresses = append(addresses, &types.Address{
			UserID:       addr.UserID,
			Address:      addr.Address,
			IsIndustrial: addr.IsIndustrial,
			IsMultisig:   addr.IsMultisig,
		})
	}

	return addresses, nil
}

func TestRightsAndAddressRightsForNominee(t *testing.T) {
	t.Parallel()

	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	nominee, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	const (
		createOp = "createEmissionApp"
		deleteOp = "deleteEmissionApp"
	)

	for _, testCase := range []struct {
		description         string
		functionName        string
		errorMsg            string
		signUser            *mocks.UserFoundation
		codeResp            int32
		funcPrepareMockStub func(t *testing.T, mockStub *mockstub.MockStub) []string
		funcCheckQuery      func(t *testing.T, mockStub *mockstub.MockStub, payload []byte)
	}{
		{
			description:  "GetRight - true",
			functionName: fnGetRight,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				mockStub.GetChannelIDReturns("cc")
				mockStub.InvokeACLMap["getAccountOperationRight"] = func(mockStub *mockstub.MockStub, parameters ...string) *peer.Response {
					if len(parameters) != acl.ArgsQtyGetAccOpRight {
						return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(parameters), acl.ArgsQtyGetAccOpRight))
					}

					rawResultData, err := proto.Marshal(&pbfound.HaveRight{HaveRight: true})
					if err != nil {
						return shim.Error(err.Error())
					}
					return shim.Success(rawResultData)
				}

				return []string{"cc", user.AddressBase58Check, acl.Issuer.String(), createOp}
			},
			funcCheckQuery: func(t *testing.T, mockStub *mockstub.MockStub, payload []byte) {
				require.Equal(t, "true", string(payload))
			},
		},
		{
			description:  "GetRight - false",
			functionName: fnGetRight,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				mockStub.GetChannelIDReturns("cc")
				mockStub.InvokeACLMap["getAccountOperationRight"] = func(mockStub *mockstub.MockStub, parameters ...string) *peer.Response {
					if len(parameters) != acl.ArgsQtyGetAccOpRight {
						return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(parameters), acl.ArgsQtyGetAccOpRight))
					}

					rawResultData, err := proto.Marshal(&pbfound.HaveRight{HaveRight: false})
					if err != nil {
						return shim.Error(err.Error())
					}
					return shim.Success(rawResultData)
				}

				return []string{"cc", user.AddressBase58Check, acl.Issuer.String(), deleteOp}
			},
			funcCheckQuery: func(t *testing.T, mockStub *mockstub.MockStub, payload []byte) {
				require.Equal(t, "false", string(payload))
			},
		},
		{
			description:  "AddressRightsForNominee - true",
			functionName: fnGetAddressRightForNominee,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				mockStub.GetChannelIDReturns("cc")
				mockStub.InvokeACLMap["getAddressRightForNominee"] = func(mockStub *mockstub.MockStub, args ...string) *peer.Response {
					if len(args) != acl.ArgsQtyGetAddressRightForNominee {
						return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(args), acl.ArgsQtyGetAddressRightForNominee))
					}

					rawResultData, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(&pbfound.HaveRight{HaveRight: true})
					if err != nil {
						return shim.Error(err.Error())
					}
					return shim.Success(rawResultData)
				}

				return []string{"cc", nominee.AddressBase58Check, user.AddressBase58Check}
			},
			funcCheckQuery: func(t *testing.T, mockStub *mockstub.MockStub, payload []byte) {
				require.Equal(t, "true", string(payload))
			},
		},
		{
			description:  "AddressRightsForNominee - false",
			functionName: fnGetAddressRightForNominee,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				mockStub.GetChannelIDReturns("cc")
				mockStub.InvokeACLMap["getAddressRightForNominee"] = func(mockStub *mockstub.MockStub, args ...string) *peer.Response {
					if len(args) != acl.ArgsQtyGetAddressRightForNominee {
						return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(args), acl.ArgsQtyGetAddressRightForNominee))
					}

					rawResultData, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(&pbfound.HaveRight{HaveRight: false})
					if err != nil {
						return shim.Error(err.Error())
					}
					return shim.Success(rawResultData)
				}

				return []string{"cc", nominee.AddressBase58Check, user.AddressBase58Check}
			},
			funcCheckQuery: func(t *testing.T, mockStub *mockstub.MockStub, payload []byte) {
				require.Equal(t, "false", string(payload))
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			mockStub.CreateAndSetConfig(
				"NT Token",
				"NT",
				8,
				issuer.AddressBase58Check,
				"",
				"",
				issuer.AddressBase58Check,
				nil,
			)

			cc, err := core.NewCC(&IssuerCheckerToken{})
			require.NoError(t, err)

			parameters := testCase.funcPrepareMockStub(t, mockStub)

			resp := mockStub.QueryChaincode(cc, testCase.functionName, parameters...)

			// check result
			require.Equal(t, resp.GetStatus(), int32(shim.OK))
			require.Empty(t, resp.GetMessage())

			if testCase.funcCheckQuery != nil {
				testCase.funcCheckQuery(t, mockStub, resp.GetPayload())
			}
		})
	}
}
