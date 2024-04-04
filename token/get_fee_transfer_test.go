package token

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/core/types/big"
	ma "github.com/anoideaopen/foundation/mock"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseToken_QueryGetFeeTransfer(t *testing.T) {
	t.Parallel()
	mock := ma.NewLedger(t)
	from := mock.NewWallet()
	to := mock.NewWallet()

	type testArgs struct {
		chaincodeArgs        FeeTransferRequestDTO
		emit                 string
		allowedBalanceToken  string
		allowedBalanceAmount uint64
	}
	tests := []struct {
		name           string
		args           testArgs
		want           *FeeTransferResponseDTO
		wantErr        assert.ErrorAssertionFunc
		wantRespMsg    string
		wantRespStatus int32
	}{
		{
			name: "success query fee of transfer",
			args: testArgs{
				chaincodeArgs: FeeTransferRequestDTO{
					SenderAddress:    from.AddressType(),
					RecipientAddress: to.AddressType(),
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
			wantErr:        assert.NoError,
			wantRespStatus: shim.OK,
			wantRespMsg:    "",
		},
		{
			name: "recipient is empty",
			args: testArgs{
				chaincodeArgs: FeeTransferRequestDTO{
					SenderAddress:    from.AddressType(),
					RecipientAddress: nil,
					Amount:           big.NewInt(10),
				},
				emit:                 "10",
				allowedBalanceToken:  "VT",
				allowedBalanceAmount: 10,
			},
			want:           nil,
			wantErr:        assert.NoError,
			wantRespStatus: shim.ERROR,
			wantRespMsg:    "failed to validate fee transfer request argument: recipient address can't be empty",
		},
		{
			name: "sender is empty",
			args: testArgs{
				chaincodeArgs: FeeTransferRequestDTO{
					SenderAddress:    nil,
					RecipientAddress: to.AddressType(),
					Amount:           big.NewInt(10),
				},
				emit:                 "10",
				allowedBalanceToken:  "VT",
				allowedBalanceAmount: 10,
			},
			want:           nil,
			wantErr:        assert.NoError,
			wantRespStatus: shim.ERROR,
			wantRespMsg:    "failed to validate fee transfer request argument: sender address can't be empty",
		},
		{
			name: "amount is empty",
			args: testArgs{
				chaincodeArgs: FeeTransferRequestDTO{
					SenderAddress:    from.AddressType(),
					RecipientAddress: to.AddressType(),
					Amount:           nil,
				},
				emit:                 "10",
				allowedBalanceToken:  "VT",
				allowedBalanceAmount: 10,
			},
			want:           nil,
			wantErr:        assert.NoError,
			wantRespStatus: shim.ERROR,
			wantRespMsg:    "failed to validate fee transfer request argument: amount must be non-negative",
		},
	}
	for testNumber, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuer := mock.NewWallet()
			feeSetter := mock.NewWallet()
			feeAddressSetter := mock.NewWallet()
			feeAggregator := mock.NewWallet()

			config := makeBaseTokenConfig("VT Token", "VT", 8,
				issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())

			name := fmt.Sprintf("vt%d", testNumber)
			mock.NewCC(name, &VT{}, config)

			issuer.SignedInvoke(name, "emitToken", tt.args.emit)
			from.AddAllowedBalance(name, tt.args.allowedBalanceToken, tt.args.allowedBalanceAmount)

			feeSetter.SignedInvoke(name, "setFee", "VT", "500000", "1", "0")
			feeAddressSetter.SignedInvoke(name, "setFeeAddress", feeAggregator.Address())

			bytes, err := json.Marshal(tt.args.chaincodeArgs)
			require.NoError(t, err)
			resp, err := from.InvokeWithPeerResponse(name, "getFeeTransfer", string(bytes))
			if !tt.wantErr(t, err, fmt.Sprintf("QueryGetFeeTransfer(%v, %v, %v)", tt.args.chaincodeArgs.SenderAddress, tt.args.chaincodeArgs.RecipientAddress, tt.args.chaincodeArgs.Amount)) {
				return
			}

			assert.Equal(t, tt.wantRespStatus, resp.Status)
			assert.Equal(t, tt.wantRespMsg, resp.Message)

			if tt.want != nil {
				feeTransferRespDTO := FeeTransferResponseDTO{}
				err = json.Unmarshal(resp.Payload, &feeTransferRespDTO)
				assert.Equal(t, tt.want.Currency, feeTransferRespDTO.Currency)
				assert.Equal(t, tt.want.Amount, feeTransferRespDTO.Amount)
				assert.Equal(t, feeAggregator.Address(), feeTransferRespDTO.FeeAddress.String())
			}
		})
	}
}
