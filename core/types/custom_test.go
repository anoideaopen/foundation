package types

import (
	"reflect"
	"testing"

	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/stretchr/testify/assert"
)

func TestConvertToAsset(t *testing.T) {
	tests := []struct {
		name    string
		in      []*MultiSwapAsset
		want    []*pb.Asset
		wantErr bool
		ErrMsg  string
	}{
		{
			name:    "check nil arg",
			in:      nil,
			want:    nil,
			wantErr: true,
			ErrMsg:  "",
		},
		{
			name:    "check empty arg",
			in:      []*MultiSwapAsset{},
			want:    []*pb.Asset{},
			wantErr: false,
			ErrMsg:  "",
		},
		{
			name: "one asset convert",
			in: []*MultiSwapAsset{
				{
					Group:  "A",
					Amount: "1",
				},
			},
			want: []*pb.Asset{
				{
					Group:  "A",
					Amount: big.NewInt(1).Bytes(),
				},
			},
			wantErr: false,
			ErrMsg:  "",
		},
		{
			name: "few asset convert",
			in: []*MultiSwapAsset{
				{
					Group:  "A",
					Amount: "1",
				},
				{
					Group:  "B",
					Amount: "2",
				},
			},
			want: []*pb.Asset{
				{
					Group:  "A",
					Amount: big.NewInt(1).Bytes(),
				},
				{
					Group:  "B",
					Amount: big.NewInt(2).Bytes(),
				},
			},
			wantErr: false,
			ErrMsg:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToAsset(tt.in)
			if (err != nil) != tt.wantErr {
				assert.EqualError(t, err, tt.ErrMsg)
				assert.Nil(t, got)
				return
			}
			if len(tt.want) == len(got) && len(got) == 0 {
				return
			}
			assert.Equal(t, len(tt.want), len(got), "size of array")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertToAsset() got = %v, want %v", got, tt.want)
			}
		})
	}
}
