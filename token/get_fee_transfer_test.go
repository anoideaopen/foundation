package token

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/stretchr/testify/require"
)

func TestBaseToken_QueryGetFeeTransfer(t *testing.T) {
	t.Parallel()

	userFrom, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	userTo, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	type testArgs struct {
		chaincodeArgs        FeeTransferRequestDTO
		emit                 string
		allowedBalanceToken  string
		allowedBalanceAmount uint64
	}
	testCollection := []struct {
		name           string
		args           testArgs
		want           *FeeTransferResponseDTO
		wantErr        require.ErrorAssertionFunc
		wantRespMsg    string
		wantRespStatus int32
	}{
		{
			name: "success query fee of transfer",
			args: testArgs{
				chaincodeArgs: FeeTransferRequestDTO{
					SenderAddress:    &types.Address{UserID: userFrom.UserID, Address: userFrom.AddressBytes},
					RecipientAddress: &types.Address{UserID: userTo.UserID, Address: userTo.AddressBytes},
					Amount:           big.NewInt(10),
				},
				emit:                 "10",
				allowedBalanceToken:  "VT",
				allowedBalanceAmount: 10,
			},
			want: &FeeTransferResponseDTO{
				Amount:   big.NewInt(1),
				Currency: "VT",
			},
			wantErr:        require.NoError,
			wantRespStatus: shim.OK,
			wantRespMsg:    "",
		},
		{
			name: "recipient is empty",
			args: testArgs{
				chaincodeArgs: FeeTransferRequestDTO{
					SenderAddress:    &types.Address{UserID: userFrom.UserID, Address: userFrom.AddressBytes},
					RecipientAddress: nil,
					Amount:           big.NewInt(10),
				},
				emit:                 "10",
				allowedBalanceToken:  "VT",
				allowedBalanceAmount: 10,
			},
			want:           nil,
			wantErr:        require.NoError,
			wantRespStatus: shim.ERROR,
			wantRespMsg:    "validation failed: 'recipient address can't be empty'",
		},
		{
			name: "sender is empty",
			args: testArgs{
				chaincodeArgs: FeeTransferRequestDTO{
					SenderAddress:    nil,
					RecipientAddress: &types.Address{UserID: userTo.UserID, Address: userTo.AddressBytes},
					Amount:           big.NewInt(10),
				},
				emit:                 "10",
				allowedBalanceToken:  "VT",
				allowedBalanceAmount: 10,
			},
			want:           nil,
			wantErr:        require.NoError,
			wantRespStatus: shim.ERROR,
			wantRespMsg:    "validation failed: 'sender address can't be empty'",
		},
		{
			name: "amount is empty",
			args: testArgs{
				chaincodeArgs: FeeTransferRequestDTO{
					SenderAddress:    &types.Address{UserID: userFrom.UserID, Address: userFrom.AddressBytes},
					RecipientAddress: &types.Address{UserID: userTo.UserID, Address: userTo.AddressBytes},
					Amount:           nil,
				},
				emit:                 "10",
				allowedBalanceToken:  "VT",
				allowedBalanceAmount: 10,
			},
			want:           nil,
			wantErr:        require.NoError,
			wantRespStatus: shim.ERROR,
			wantRespMsg:    "validation failed: 'amount must be non-negative'",
		},
	}
	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			feeSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			feeAddressSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			feeAggregator, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			mockStub.CreateAndSetConfig(
				"VT Token",
				"VT",
				8,
				issuer.AddressBase58Check,
				feeSetter.AddressBase58Check,
				feeAddressSetter.AddressBase58Check,
				"",
				nil,
			)

			cc, err := core.NewCC(&VT{})
			require.NoError(t, err)

			// preparing mockStub
			user1BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{userFrom.AddressBase58Check, test.args.allowedBalanceToken})
			require.NoError(t, err)

			metadata := &pbfound.Token{
				Fee: &pbfound.TokenFee{
					Currency: "VT",
					Fee:      big.NewInt(500000).Bytes(),
					Floor:    big.NewInt(1).Bytes(),
					Cap:      big.NewInt(0).Bytes(),
				},
				FeeAddress: feeAggregator.AddressBytes,
			}

			rawMetadata, err := proto.Marshal(metadata)
			require.NoError(t, err)

			mockStub.GetStateCallsMap[user1BalanceKey] = new(big.Int).SetUint64(test.args.allowedBalanceAmount).Bytes()
			mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

			bytes, err := json.Marshal(test.args.chaincodeArgs)
			require.NoError(t, err)

			resp := mockStub.QueryChaincode(cc, "getFeeTransfer", []string{string(bytes)}...)
			test.wantErr(t, err, fmt.Sprintf("QueryGetFeeTransfer(%v, %v, %v)", test.args.chaincodeArgs.SenderAddress, test.args.chaincodeArgs.RecipientAddress, test.args.chaincodeArgs.Amount))

			require.Equal(t, test.wantRespStatus, resp.Status)
			require.Contains(t, resp.Message, test.wantRespMsg)

			if test.want != nil {
				feeTransferRespDTO := FeeTransferResponseDTO{}
				_ = json.Unmarshal(resp.Payload, &feeTransferRespDTO)
				require.Equal(t, test.want.Currency, feeTransferRespDTO.Currency)
				require.Equal(t, test.want.Amount, feeTransferRespDTO.Amount)
				require.Equal(t, feeAggregator.AddressBase58Check, feeTransferRespDTO.FeeAddress.String())
			}
		})
	}
}
