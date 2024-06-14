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
	"github.com/stretchr/testify/require"
)

type TestMultiSwapAsset struct {
	Group   string       `json:"group,omitempty"`
	Amount  *corebig.Int `json:"amount,omitempty"`
	Amount2 *big.Int     `json:"amount2,omitempty"`
}

type TestMultiSwapAssets struct {
	Assets []*TestMultiSwapAsset `json:"assets,omitempty"`
}

type TestStructForCall struct{}

func (t *TestStructForCall) Method1(ts *time.Time) {
	fmt.Printf("ts: %v\n", ts)
}

func (t *TestStructForCall) Method2(ts time.Time) {
	fmt.Printf("ts: %v\n", ts)
}

func (t *TestStructForCall) Method3(a *proto.Address) string {
	fmt.Printf("a: %+v\n", a)
	return a.AddrString()
}

func (t *TestStructForCall) Method4(in float64) {
	fmt.Printf("in: %+v\n", in)
}

func (t *TestStructForCall) Method5(in []float64) {
	fmt.Printf("in: %+v\n", in)
}

func (t *TestStructForCall) Method6(in *big.Int) {
	fmt.Printf("in: %+v\n", in)
}

func (t *TestStructForCall) Method7(in string) {
	fmt.Printf("in: %+v\n", in)
}

func (t *TestStructForCall) Method8(in *string) {
	fmt.Printf("in: %+v\n", in)
}

func (t *TestStructForCall) Method9(in *int) {
	fmt.Printf("in: %+v\n", in)
}

func (t *TestStructForCall) Method10(in types.MultiSwapAssets) types.MultiSwapAssets {
	for _, a := range in.Assets {
		fmt.Printf("a: %+v\n", a)
	}
	return in
}

func (t *TestStructForCall) Method11(in *types.MultiSwapAssets) *types.MultiSwapAssets {
	for _, a := range in.Assets {
		fmt.Printf("a: %+v\n", a)
	}
	return in
}

func (t *TestStructForCall) Method12(in TestMultiSwapAssets) TestMultiSwapAssets {
	for _, a := range in.Assets {
		fmt.Printf("a: %+v\n", a)
	}
	return in
}

func (t *TestStructForCall) Method13(in *TestMultiSwapAssets) *TestMultiSwapAssets {
	for _, a := range in.Assets {
		fmt.Printf("a: %+v\n", a)
	}
	return in
}

func TestCall(t *testing.T) {
	input := &TestStructForCall{}

	a := &proto.Address{
		UserID:       "1234",
		Address:      []byte{1, 2, 3, 4},
		IsIndustrial: true,
		IsMultisig:   false,
	}
	aJSON, _ := json.Marshal(a)

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
		name      string
		method    string
		args      []string
		wantLen   int
		wantErr   bool
		wantValue any
	}{
		{
			name:    "MethodX unsupported method",
			method:  "MethodX",
			args:    []string{},
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "Method1 with correct time format",
			method:  "Method1",
			args:    []string{time.Now().Format(time.RFC3339)},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "Method1 with correct binary time format",
			method:  "Method1",
			args:    []string{string(nowBinary)},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "Method2 with correct time format",
			method:  "Method2",
			args:    []string{time.Now().Format(time.RFC3339)},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:      "Method3 with JSON",
			method:    "Method3",
			args:      []string{string(aJSON)},
			wantLen:   1,
			wantErr:   false,
			wantValue: a.AddrString(),
		},
		{
			name:    "Method4 with float input",
			method:  "Method4",
			args:    []string{"1234.5678"},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "Method5 with array input",
			method:  "Method5",
			args:    []string{"[1234.5678, 1234.5678]"},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "Method5 with incorrect format",
			method:  "Method5",
			args:    []string{"1234.5678, 1234.5678"},
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "Method5 with incorrect args count",
			method:  "Method5",
			args:    []string{"1234.5678", "1234.5678"},
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "Method6 with big.Int input",
			method:  "Method6",
			args:    []string{"1234"},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "Method6 with incorrect value type big.Int",
			method:  "Method6",
			args:    []string{"1234.5678"},
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "Method7 with string input",
			method:  "Method7",
			args:    []string{"1234"},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "Method8 with string input",
			method:  "Method8",
			args:    []string{"1234"},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "Method9 with int input",
			method:  "Method9",
			args:    []string{"1234"},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:      "Method10 with a complex MultiSwapAssets input and output",
			method:    "Method10",
			args:      []string{string(multiSwapAssetsJSON)},
			wantLen:   1,
			wantErr:   false,
			wantValue: multiSwapAssets,
		},
		{
			name:      "Method11 with a complex MultiSwapAssets input and output",
			method:    "Method11",
			args:      []string{string(multiSwapAssetsJSON)},
			wantLen:   1,
			wantErr:   false,
			wantValue: &multiSwapAssets,
		},
		{
			name:      "Method12 with a complex TestMultiSwapAssets input and output",
			method:    "Method12",
			args:      []string{string(testMultiSwapAssetsJSON)},
			wantLen:   1,
			wantErr:   false,
			wantValue: testMultiSwapAssets,
		},
		{
			name:      "Method13 with a complex TestMultiSwapAssets input and output",
			method:    "Method13",
			args:      []string{string(testMultiSwapAssetsJSON)},
			wantLen:   1,
			wantErr:   false,
			wantValue: &testMultiSwapAssets,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := Call(input, tt.method, tt.args...)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, resp, tt.wantLen)
			if tt.wantValue != nil {
				require.Equal(t, tt.wantValue, resp[0])
			}
		})
	}
}
