package unit

import (
	"encoding/json"
	"math/big"
	"testing"

	customBig "github.com/anoideaopen/foundation/core/types/big"
	"github.com/stretchr/testify/require"
)

type BigInt struct {
	NumberField *big.Int `json:"num"`
}

type CustomBigInt struct {
	NumberField *customBig.Int `json:"num"`
}

const bigIntVal = "111111111111111111111111111111111111111111111111111111" +
	"111111111111111111111111111111111111111111111111111111111111120"

// TestStdBigIntStructMarshalling checks serialization|deserialization of std *big.Int type
func TestStdBigIntStructMarshalling(t *testing.T) {
	tbiStr := "{\"num\":" + bigIntVal + "}"

	bi, ok := new(big.Int).SetString(bigIntVal, 10)
	require.Equal(t, true, ok)
	tbi := BigInt{
		NumberField: bi,
	}

	t.Run("struct with *big.Int marshal test", func(t *testing.T) {
		tbiData, err := json.Marshal(tbi)
		require.NoError(t, err)
		require.Equal(t, tbiStr, string(tbiData))
	})

	t.Run("struct with *big.Int unmarshall test", func(t *testing.T) {
		var tbi1 BigInt
		err := json.Unmarshal([]byte(tbiStr), &tbi1)
		require.NoError(t, err)
		require.Equal(t, tbi, tbi1)
	})
}

// TestCustomBigIntStructMarshalling checks serialization|deserialization of std *big.Int type.
// This custom type was added because NodeJS backend can't work with *big.Int when it converted as json.number.
// So now, all big.Int's are converted to string type. But other int's are real ints.
func TestCustomBigIntStructMarshalling(t *testing.T) {
	tbiStr := "{\"num\":\"" + bigIntVal + "\"}" // added \" quotes

	bi, ok := new(customBig.Int).SetString(bigIntVal, 10)
	require.Equal(t, true, ok)
	tbi := CustomBigInt{
		NumberField: bi,
	}

	t.Run("struct with *big.Int marshal test", func(t *testing.T) {
		tbiData, err := json.Marshal(tbi)
		require.NoError(t, err)
		require.Equal(t, tbiStr, string(tbiData))
	})

	t.Run("struct with *big.Int unmarshall test", func(t *testing.T) {
		var tbi1 CustomBigInt
		err := json.Unmarshal([]byte(tbiStr), &tbi1)
		require.NoError(t, err)
		require.Equal(t, tbi, tbi1)
	})
}
