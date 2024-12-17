package mocks

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

const TestCreatorMSP = "platformMSP"

func SetCreator(mockStub *ChaincodeStub, certString string) error {
	certificate, err := hex.DecodeString(certString)
	if err != nil {
		return err
	}
	mockStub.GetCreatorReturns(certificate, nil)
	return nil
}

func SetCreatorCert(mockStub *ChaincodeStub, msp string, cert string) error {
	certificate, _ := base64.StdEncoding.DecodeString(cert)
	creator, err := MarshalIdentity(msp, certificate)
	if err != nil {
		return err
	}
	mockStub.GetCreatorReturns(creator, nil)
	return nil
}

func MarshalIdentity(creatorMSP string, creatorCert []byte) ([]byte, error) {
	pemblock := &pem.Block{Type: "CERTIFICATE", Bytes: creatorCert}
	pemBytes := pem.EncodeToMemory(pemblock)
	if pemBytes == nil {
		return nil, errors.New("encoding of identity failed")
	}

	creator := &msp.SerializedIdentity{Mspid: creatorMSP, IdBytes: pemBytes}
	marshaledIdentity, err := proto.Marshal(creator)
	if err != nil {
		return nil, err
	}
	return marshaledIdentity, nil
}

// GetNewStringNonce returns string value of nonce based on current time
func GetNewStringNonce() string {
	return strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
}

// SetFunctionAndParameters sets function name and its parameters to ChaincodeStub
func SetFunctionAndParameters(mockStub *ChaincodeStub, functionName, requestID, channelName, chaincodeName string, args ...string) {
	ctorArgs := append(append([]string{requestID, channelName, chaincodeName}, args...), GetNewStringNonce())
	mockStub.GetFunctionAndParametersReturns(functionName, ctorArgs)
}

// SetFunctionAndParametersWithSign sets function name and parameters with sender sign to ChaincodeStub
func SetFunctionAndParametersWithSign(
	mockStub *ChaincodeStub,
	user *UserFoundation,
	functionName,
	requestID,
	channelName,
	chaincodeName string,
	args ...string,
) error {
	ctorArgs := append(append([]string{functionName, requestID, channelName, chaincodeName}, args...), GetNewStringNonce())

	pubKey, sMsg, err := user.Sign(ctorArgs...)
	if err != nil {
		return err
	}

	mockStub.GetFunctionAndParametersReturns(functionName, append(ctorArgs[1:], pubKey, base58.Encode(sMsg)))
	return nil
}

// ACLGetAccountInfo mocks positive response from ACL
func ACLGetAccountInfo(t *testing.T, mockStub *ChaincodeStub, invokeCallCount int) {
	accInfo := &pbfound.AccountInfo{}

	rawAccInfo, err := json.Marshal(accInfo)
	require.NoError(t, err)

	// mock acl response
	mockStub.InvokeChaincodeReturnsOnCall(invokeCallCount, peer.Response{
		Status:  http.StatusOK,
		Message: "",
		Payload: rawAccInfo,
	})
}

// ACLCheckSigner mocks positive response from ACL for signer
func ACLCheckSigner(t *testing.T, mockStub *ChaincodeStub, user *UserFoundation, isIndustrial bool) {
	userAddress := sha3.Sum256(user.PublicKeyBytes)

	aclResponse := &pbfound.AclResponse{
		Account: &pbfound.AccountInfo{},
		Address: &pbfound.SignedAddress{
			Address: &pbfound.Address{
				UserID:       user.UserID,
				Address:      userAddress[:],
				IsIndustrial: isIndustrial,
			},
		},
		KeyTypes: []pbfound.KeyType{user.KeyType},
	}

	rawResponse, err := proto.Marshal(aclResponse)
	require.NoError(t, err)

	mockStub.InvokeChaincodeReturnsOnCall(0, peer.Response{
		Status:  http.StatusOK,
		Message: "",
		Payload: rawResponse,
	})
}
