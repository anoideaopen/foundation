package core

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/types"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

// CheckSign checks the signature of the transaction
func CheckSign(stub shim.ChaincodeStubInterface, fn string, args []string, auth []string) (*types.Address, string, error) {
	signers := len(auth) / 2 //nolint:gomnd
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

func (cc *ChainCode) checkAuthIfNeeds( //nolint:gocognit,funlen
	stub shim.ChaincodeStubInterface,
	method *Fn,
	fn string,
	args []string,
	_ bool, // check
) (*pb.Address, []string, uint64, error) {
	if !method.needsAuth {
		return nil, args, 0, nil
	}
	total := len(args)
	argMethodLen := len(method.in)
	// requestID := args[0]
	chaincodeName := args[1]
	channelName := args[2]
	nonceStr := args[3+argMethodLen]
	// c authPos is a list of who signed, after their signature
	// the number of signatories and signatures must be equal
	authPos := argMethodLen + 4 //nolint:gomnd    // + reqId - 0, cc - 1, ch - 2, nonce - argMethodLen+3

	if total < authPos {
		return nil, nil, 0, fmt.Errorf("incorrect number of arguments. found %d but expected more or eq %d",
			total, authPos)
	}

	spr, err := stub.GetSignedProposal()
	if err != nil {
		return nil, nil, 0, err
	}
	proposal := &peer.Proposal{}
	if err = proto.Unmarshal(spr.ProposalBytes, proposal); err != nil {
		return nil, nil, 0, err
	}
	payload := &peer.ChaincodeProposalPayload{}
	if err = proto.Unmarshal(proposal.Payload, payload); err != nil {
		return nil, nil, 0, err
	}
	input := &peer.ChaincodeInvocationSpec{}
	if err = proto.Unmarshal(payload.Input, input); err != nil {
		return nil, nil, 0, err
	}

	if input.ChaincodeSpec == nil ||
		input.ChaincodeSpec.ChaincodeId == nil ||
		chaincodeName != input.ChaincodeSpec.ChaincodeId.Name {
		return nil, nil, 0, fmt.Errorf("incorrect chaincode name in args by index 1. found %s but expected %s",
			chaincodeName, input.ChaincodeSpec.ChaincodeId.Name)
	}

	if channelName != stub.GetChannelID() {
		return nil, nil, 0, fmt.Errorf("incorrect channel name in args by index 2. found %s but expected %s",
			channelName, stub.GetChannelID())
	}

	if len(args[authPos:])%2 != 0 {
		return nil, nil, 0, errors.New("incorrect number of keys or signs")
	}

	signers := (total - authPos) / 2 //nolint:gomnd
	if signers == 0 {
		return nil, nil, 0, errors.New("should be signed")
	}

	message := sha3.Sum256([]byte(fn + strings.Join(args[:len(args)-signers], "")))

	acl, err := helpers.CheckACL(stub, args[authPos:authPos+signers])
	if err != nil {
		return nil, nil, 0, err
	}
	N := 1 // for single sign
	if signers > 1 {
		if acl.Address != nil && acl.Address.SignaturePolicy != nil {
			N = int(acl.Address.SignaturePolicy.N)
		} else {
			N = signers // If it's not in the acl, everyone has to sign
		}
	}

	for i := authPos; i < authPos+signers; i++ {
		if args[i+signers] == "" {
			continue
		}
		key := base58.Decode(args[i])
		sign := base58.Decode(args[i+signers])
		if len(key) != ed25519.PublicKeySize || !ed25519.Verify(key, message[:], sign) {
			return nil, nil, 0, errors.New("incorrect signature")
		}

		N--
	}

	if N > 0 {
		return nil, nil, 0, errors.New("signature policy isn't satisfied")
	}

	if acl.Account != nil && acl.Account.BlackListed {
		return nil, nil, 0, fmt.Errorf("address %s is blacklisted", (*types.Address)(acl.Address.Address).String())
	}
	if acl.Account != nil && acl.Account.GrayListed {
		return nil, nil, 0, fmt.Errorf("address %s is graylisted", (*types.Address)(acl.Address.Address).String())
	}

	if err = helpers.AddAddrIfChanged(stub, acl.Address); err != nil {
		return nil, nil, 0, err
	}

	nonce, err := strconv.ParseUint(nonceStr, 10, 64)
	if err != nil {
		return nil, nil, 0, err
	}

	// Let's run the nonce the old-fashioned way
	if cc.nonceTTL == 0 {
		if err = cc.nonceCheckFn(stub, types.NewSenderFromAddr((*types.Address)(acl.Address.Address)), nonce); err != nil {
			return nil, nil, 0, fmt.Errorf("incorrect nonce: %w", err)
		}
	}

	return acl.Address.Address, args[3 : 3+argMethodLen], nonce, nil
}
