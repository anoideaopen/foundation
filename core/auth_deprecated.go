//nolint:gomnd
package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/keys"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
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

	message := []byte(fn + strings.Join(append(args, auth[:signers]...), ""))
	for i := 0; i < signers; i++ {
		key := base58.Decode(auth[i])
		sign := base58.Decode(auth[i+signers])
		valid, err := keys.VerifySignatureByKeyType(pb.KeyType_ed25519, key, message, sign)
		if err != nil {
			return &types.Address{}, "", fmt.Errorf("error validating signature: %w", err)
		}
		if !valid {
			return &types.Address{}, "", errors.New("incorrect signature")
		}
	}

	acl, err := helpers.CheckACL(stub, auth[:signers])
	if err != nil {
		return &types.Address{}, "", err
	}

	if acl.GetAccount().GetGrayListed() {
		errMsg := fmt.Sprintf("address %s is graylisted", (*types.Address)(acl.GetAddress().GetAddress()).String())
		return &types.Address{}, "", errors.New(errMsg)
	}

	hash := sha3.Sum256(message)
	return (*types.Address)(acl.GetAddress().GetAddress()), hex.EncodeToString(hash[:]), nil
}
