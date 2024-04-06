//nolint:gomnd
package core

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

// CheckSign exists fo backward compatibility
func CheckSign(
	stub shim.ChaincodeStubInterface,
	fn string,
	args []string,
	auth []string,
) (*types.Address, string, error) {
	signers := len(auth) / 2
	if signers == 0 {
		return &types.Address{}, "", errors.New("should be signed")
	}

	message := sha3.Sum256([]byte(fn + strings.Join(append(args, auth[:signers]...), "")))
	for i := 0; i < signers; i++ {
		key := base58.Decode(auth[i])
		sign := base58.Decode(auth[i+signers])
		if !ed25519.Verify(key, message[:], sign) {
			return &types.Address{}, "", errors.New("incorrect signature")
		}
	}

	acl, err := helpers.CheckACL(stub, auth[:signers])
	if err != nil {
		return &types.Address{}, "", err
	}

	if acl.Account != nil && acl.Account.GrayListed {
		errMsg := fmt.Sprintf("address %s is graylisted", (*types.Address)(acl.Address.Address).String())
		return &types.Address{}, "", fmt.Errorf(errMsg)
	}

	return (*types.Address)(acl.Address.Address), hex.EncodeToString(message[:]), nil
}
