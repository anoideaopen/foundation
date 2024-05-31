package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/anoideaopen/foundation/core/contract"
	"github.com/anoideaopen/foundation/core/gost"
	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/types"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ddulesov/gogost/gost3410"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

type invocationDetails struct {
	chaincodeNameArg string
	channelNameArg   string
	nonceStringArg   string
	signatureArgs    []string
	signersCount     int
	keyTypes         []pb.KeyType
}

// validateAndExtractInvocationContext verifies authorization and extracts the context of the chincode method call.
// This function makes sure that the number of arguments matches the expected number of arguments,
// verifies that the chancode name and channel match, authenticates signatures,
// updates the address if necessary, and verifies the nonce.
// Returns the user's address, a list of method arguments and nonce if successful, or an error.
//
// Parameters:
//   - stub - interface to interact with the blockchain.
//   - fnMetadata - metadata of the called method.
//   - fn - name of the called method.
//   - args - arguments of the call.
//
// Return values:
//   - User address, method call arguments, nonce and error, if any.
func (cc *Chaincode) validateAndExtractInvocationContext(
	stub shim.ChaincodeStubInterface,
	method contract.Method,
	args []string,
) (sender *pb.Address, invocationArgs []string, nonce uint64, err error) {
	// If authorization is not required, return the arguments unchanged.
	if !method.RequiresAuth {
		return nil, args, 0, nil
	}

	invocation, err := parseInvocationDetails(method, args)
	if err != nil {
		return nil, nil, 0, err
	}

	// Check the correspondence between the name and the channel of the chancode.
	if err = checkChaincodeAndChannelName(
		stub,
		invocation.chaincodeNameArg,
		invocation.channelNameArg,
	); err != nil {
		return nil, nil, 0, err
	}

	signers := invocation.signatureArgs[:invocation.signersCount]

	// Check the ACL (access control list).
	acl, err := checkACLSignerStatus(stub, signers)
	if err != nil {
		return nil, nil, 0, err
	}

	oldBehavior := invocation.signersCount != len(acl.GetKeyTypes())
	invocation.keyTypes = make([]pb.KeyType, len(signers))
	for i := 0; i < invocation.signersCount; i++ {
		if oldBehavior {
			if len(signers[i]) == int(gost3410.Mode2012) {
				invocation.keyTypes[i] = pb.KeyType_gost
			} else {
				invocation.keyTypes[i] = pb.KeyType_ed25519
			}
		} else {
			invocation.keyTypes[i] = acl.GetKeyTypes()[i]
		}
	}

	// Form a message to verify the signature.
	message := []byte(method.ChaincodeFunc + strings.Join(args[:len(args)-invocation.signersCount], ""))

	if err = validateSignaturesInInvocation(invocation, message); err != nil {
		return nil, nil, 0, err
	}

	// Update the address if it has changed.
	if err = helpers.AddAddrIfChanged(stub, acl.GetAddress()); err != nil {
		return nil, nil, 0, err
	}

	// Convert nonce from a string to a number.
	nonce, err = strconv.ParseUint(invocation.nonceStringArg, 10, 64)
	if err != nil {
		return nil, nil, 0, err
	}

	// Return the signer's address, method arguments, and nonce.
	return acl.GetAddress().GetAddress(), args[3 : 3+(method.NumArgs-1)], nonce, nil
}

func validateSignaturesInInvocation(
	invocation *invocationDetails,
	message []byte,
) error {
	var (
		digestSHA3 []byte
		digestGOST []byte
		err        error
	)

	for i := 0; i < invocation.signersCount; i++ {
		if invocation.signatureArgs[i+invocation.signersCount] == "" {
			continue // Skip the blank signatures.
		}

		var (
			publicKey = base58.Decode(invocation.signatureArgs[i])
			signature = base58.Decode(invocation.signatureArgs[i+invocation.signersCount])
		)

		// Verify the signature ED25519, ECDSA or GOST 34.10 2012
		valid := false
		switch invocation.keyTypes[i] {
		case pb.KeyType_ecdsa:
			if digestSHA3 == nil {
				digestSHA3Raw := sha3.Sum256(message)
				digestSHA3 = digestSHA3Raw[:]
			}
			ecdsaKey := &ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     new(big.Int).SetBytes(publicKey[:32]),
				Y:     new(big.Int).SetBytes(publicKey[32:]),
			}
			valid = ecdsa.VerifyASN1(ecdsaKey, digestSHA3, signature)
		case pb.KeyType_gost:
			if digestGOST == nil {
				digestGOSTRaw := gost.Sum256(message)
				digestGOST = digestGOSTRaw[:]
			}
			valid, err = gost.Verify(publicKey, digestGOST, signature)
			if err != nil {
				return fmt.Errorf("incorrect signature: %w", err)
			}
		default:
			if digestSHA3 == nil {
				digestSHA3Raw := sha3.Sum256(message)
				digestSHA3 = digestSHA3Raw[:]
			}
			valid = ed25519.Verify(publicKey, digestSHA3, signature)
		}

		if !valid {
			return errors.New("incorrect signature")
		}
	}
	return nil
}

func checkACLSignerStatus(stub shim.ChaincodeStubInterface, signers []string) (*pb.AclResponse, error) {
	acl, err := helpers.CheckACL(stub, signers)
	if err != nil {
		return nil, err
	}

	// Check the status of the signer in the access control list.
	if acl.GetAccount() != nil {
		if acl.GetAccount().GetBlackListed() {
			return nil, fmt.Errorf("address %s is blacklisted", (*types.Address)(acl.GetAddress().GetAddress()).String())
		}
		if acl.GetAccount().GetGrayListed() {
			return nil, fmt.Errorf("address %s is graylisted", (*types.Address)(acl.GetAddress().GetAddress()).String())
		}
	}

	return acl, nil
}

func parseInvocationDetails(
	method contract.Method,
	args []string,
) (*invocationDetails, error) {
	// Calculating the positions of arguments in an array.
	var (
		expectedArgsCount = (method.NumArgs - 1) + 4 // +4 for reqId, cc, ch, nonce
		authArgsStartPos  = expectedArgsCount        // Authorization arguments start position
	)

	// We check that the number of arguments is not less than expected.
	if len(args) < expectedArgsCount {
		return nil, fmt.Errorf(
			"incorrect number of arguments. found %d but expected more or eq %d",
			len(args),
			expectedArgsCount,
		)
	}

	// Check that the number of keys and signatures is correct.
	if len(args[authArgsStartPos:])%2 != 0 {
		return nil, fmt.Errorf(
			"incorrect number of keys or signs. signs started at: %d in args: %v",
			authArgsStartPos,
			args,
		)
	}

	signersCount := (len(args) - authArgsStartPos) / 2
	if signersCount == 0 {
		return nil, errors.New("should be signed")
	}

	// Extracting the main arguments.
	basicArgsData := &invocationDetails{
		chaincodeNameArg: args[1],
		channelNameArg:   args[2],
		nonceStringArg:   args[authArgsStartPos-1],
		signersCount:     signersCount,
		signatureArgs:    args[authArgsStartPos : authArgsStartPos+signersCount*2],
	}

	return basicArgsData, nil
}

func checkChaincodeAndChannelName(
	stub shim.ChaincodeStubInterface,
	chaincodeName string,
	channelName string,
) error {
	// Getting the offer of a signature.
	signedProposal, err := stub.GetSignedProposal()
	if err != nil {
		return err
	}

	proposal := &peer.Proposal{}
	if err = proto.Unmarshal(signedProposal.GetProposalBytes(), proposal); err != nil {
		return err
	}

	payload := &peer.ChaincodeProposalPayload{}
	if err = proto.Unmarshal(proposal.GetPayload(), payload); err != nil {
		return err
	}

	invocationSpec := &peer.ChaincodeInvocationSpec{}
	if err = proto.Unmarshal(payload.GetInput(), invocationSpec); err != nil {
		return err
	}

	// Check the correspondence between the name and the channel of the chancode.
	if chaincodeName != invocationSpec.GetChaincodeSpec().GetChaincodeId().GetName() {
		return fmt.Errorf(
			"incorrect chaincode name in args by index 1. found %s but expected %s",
			chaincodeName,
			invocationSpec.GetChaincodeSpec().GetChaincodeId().GetName(),
		)
	}

	if channelName != stub.GetChannelID() {
		return fmt.Errorf(
			"incorrect channel name in args by index 2. found %s but expected %s",
			channelName,
			stub.GetChannelID(),
		)
	}

	return nil
}
