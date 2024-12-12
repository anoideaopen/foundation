package unit

import (
	"crypto/ecdsa"
	"testing"

	"github.com/anoideaopen/foundation/keys/eth"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_KeysEth(t *testing.T) {
	const (
		messageHex       = "0xb412a9afc250a81b76a64bf59f960839489577ccc5a9b545c574de11a2769455"
		privateKeyBase58 = "C9esAjsYJEhaTvfMRrPcFnY2WLnmTdohvVzd8dzxPZ3v"
		publicKeyBase58  = "PmNVcznMPM7xg5eSGWA7LLrW2kqfNMbnpEBVWhKg3yGShfEj6Eec5KrahQFTWBuQQ8ZHecPtXVCUm88ensE6ztKG"

		expectedMessageHashHex = "0x5bfd8fe42a24d57342ac211dcf319ec148302c17b0f0bfa85d83fb82bb13ac5b"
		expectedSignatureHex   = "0xf39b93ed322d7334c891516d8bee70b44c6b46b2dc3b9f6ad06d896975ffca0511f712296cc705d435cff51391275f8ae3dd09a4d5619df7a295606cc8e555d21c"
	)

	var (
		digest     []byte
		signature  []byte
		privateKey *ecdsa.PrivateKey
	)

	t.Run("ethereum hash", func(t *testing.T) {
		var (
			message  = hexutil.MustDecode(messageHex)
			expected = hexutil.MustDecode(expectedMessageHashHex)
		)
		digest = eth.Hash(message)
		assert.Equal(t, expected, digest)
	})

	t.Run("ethereum signature", func(t *testing.T) {
		var (
			err      error
			expected = hexutil.MustDecode(expectedSignatureHex)
		)
		privateKey, err = eth.PrivateKeyFromBytes(base58.Decode(privateKeyBase58))
		require.NoError(t, err)
		signature, err = eth.Sign(digest, privateKey)
		require.NoError(t, err)
		assert.Equal(t, expected, signature)
	})

	t.Run("verify ethereum signature", func(t *testing.T) {
		publicKey := base58.Decode(publicKeyBase58)
		assert.True(t, eth.Verify(publicKey, digest, signature))
	})
}
