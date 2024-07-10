package reflectx

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core/types"
	corebig "github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

type TestStructForValidation struct{}

func (t *TestStructForValidation) Method1(ts *time.Time) {
	// Example method
}

func (t *TestStructForValidation) Method2(ts time.Time) {
	// Example method
}

func (t *TestStructForValidation) Method3(a *proto.Address) string {
	return a.AddrString()
}

func (t *TestStructForValidation) Method4(in float64) {
	// Example method
}

func (t *TestStructForValidation) Method5(in []float64) {
	// Example method
}

func (t *TestStructForValidation) Method6(in *big.Int) {
	// Example method
}

func (t *TestStructForValidation) Method7(in string) {
	// Example method
}

func (t *TestStructForValidation) Method8(in *string) {
	// Example method
}

func (t *TestStructForValidation) Method9(in *int) {
	// Example method
}

func (t *TestStructForValidation) Method10(in types.MultiSwapAssets) types.MultiSwapAssets {
	return in
}

func (t *TestStructForValidation) Method11(in *types.MultiSwapAssets) *types.MultiSwapAssets {
	return in
}

func (t *TestStructForValidation) Method12(in TestMultiSwapAssets) TestMultiSwapAssets {
	return in
}

func (t *TestStructForValidation) Method13(in *TestMultiSwapAssets) *TestMultiSwapAssets {
	return in
}

type mockStub struct {
	shim.ChaincodeStubInterface
}

type TestValidator struct {
	Value string
}

func (v *TestValidator) Check() error {
	if v.Value == "" {
		return fmt.Errorf("invalid value")
	}
	return nil
}

func (t *TestStructForValidation) Method14(in *TestValidator) {
	// Example method
}

func TestValidateArguments(t *testing.T) {
	input := &TestStructForValidation{}
	stub := &mockStub{}

	a := &proto.Address{
		UserID:       "1234",
		Address:      []byte{1, 2, 3, 4},
		IsIndustrial: true,
		IsMultisig:   false,
	}
	aJSON, _ := protojson.Marshal(a)
	nowBinary, _ := time.Now().MarshalBinary()

	multiSwapAssets := types.MultiSwapAssets{
		Assets: []*types.MultiSwapAsset{
			{
				Group:  "A",
				Amount: "1",
			},
			{
				Group:  "B",
				Amount: "2",
			},
		},
	}
	multiSwapAssetsJSON, _ := json.Marshal(multiSwapAssets)

	testMultiSwapAssets := TestMultiSwapAssets{
		Assets: []*TestMultiSwapAsset{
			{
				Group:   "C",
				Amount:  corebig.NewInt(1),
				Amount2: big.NewInt(2),
			},
			{
				Group:   "D",
				Amount:  corebig.NewInt(3),
				Amount2: big.NewInt(4),
			},
		},
	}
	testMultiSwapAssetsJSON, _ := json.Marshal(testMultiSwapAssets)

	tests := []struct {
		name    string
		method  string
		args    []string
		wantErr bool
	}{
		{
			name:    "MethodX unsupported method",
			method:  "MethodX",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "Method1 with correct time format",
			method:  "Method1",
			args:    []string{time.Now().Format(time.RFC3339)},
			wantErr: false,
		},
		{
			name:    "Method1 with correct binary time format",
			method:  "Method1",
			args:    []string{string(nowBinary)},
			wantErr: false,
		},
		{
			name:    "Method2 with correct time format",
			method:  "Method2",
			args:    []string{time.Now().Format(time.RFC3339)},
			wantErr: false,
		},
		{
			name:    "Method3 with JSON",
			method:  "Method3",
			args:    []string{string(aJSON)},
			wantErr: false,
		},
		{
			name:    "Method4 with float input",
			method:  "Method4",
			args:    []string{"1234.5678"},
			wantErr: false,
		},
		{
			name:    "Method5 with array input",
			method:  "Method5",
			args:    []string{"[1234.5678, 1234.5678]"},
			wantErr: false,
		},
		{
			name:    "Method5 with incorrect format",
			method:  "Method5",
			args:    []string{"1234.5678, 1234.5678"},
			wantErr: true,
		},
		{
			name:    "Method5 with incorrect args count",
			method:  "Method5",
			args:    []string{"1234.5678", "1234.5678"},
			wantErr: true,
		},
		{
			name:    "Method6 with big.Int input",
			method:  "Method6",
			args:    []string{"1234"},
			wantErr: false,
		},
		{
			name:    "Method6 with incorrect value type big.Int",
			method:  "Method6",
			args:    []string{"1234.5678"},
			wantErr: true,
		},
		{
			name:    "Method7 with string input",
			method:  "Method7",
			args:    []string{"1234"},
			wantErr: false,
		},
		{
			name:    "Method8 with string input",
			method:  "Method8",
			args:    []string{"1234"},
			wantErr: false,
		},
		{
			name:    "Method9 with int input",
			method:  "Method9",
			args:    []string{"1234"},
			wantErr: false,
		},
		{
			name:    "Method10 with a complex MultiSwapAssets input",
			method:  "Method10",
			args:    []string{string(multiSwapAssetsJSON)},
			wantErr: false,
		},
		{
			name:    "Method11 with a complex MultiSwapAssets input",
			method:  "Method11",
			args:    []string{string(multiSwapAssetsJSON)},
			wantErr: false,
		},
		{
			name:    "Method12 with a complex TestMultiSwapAssets input",
			method:  "Method12",
			args:    []string{string(testMultiSwapAssetsJSON)},
			wantErr: false,
		},
		{
			name:    "Method13 with a complex TestMultiSwapAssets input",
			method:  "Method13",
			args:    []string{string(testMultiSwapAssetsJSON)},
			wantErr: false,
		},
		{
			name:    "Method14 with valid TestValidator",
			method:  "Method14",
			args:    []string{`{"Value": "valid"}`},
			wantErr: false,
		},
		{
			name:    "Method14 with invalid TestValidator",
			method:  "Method14",
			args:    []string{`{"Value": ""}`},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArguments(input, tt.method, stub, tt.args...)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
