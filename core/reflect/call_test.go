package reflect

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"
)

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

func TestCall(t *testing.T) {
	input := &TestStructForCall{}

	a := &proto.Address{
		UserID:       "1234",
		Address:      []byte{1, 2, 3, 4},
		IsIndustrial: true,
		IsMultisig:   false,
	}
	aJSON, _ := protojson.Marshal(a)
	aRaw, _ := pb.Marshal(a)

	nowBinary, _ := time.Now().MarshalBinary()
	valueGOB, _ := big.NewInt(42).GobEncode()

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
			name:    "Method3 with Protobuf",
			method:  "Method3",
			args:    []string{string(aRaw)},
			wantLen: 1,
			wantErr: false,
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
			name:    "Method6 with GOB input",
			method:  "Method6",
			args:    []string{string(valueGOB)},
			wantLen: 0,
			wantErr: false,
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
