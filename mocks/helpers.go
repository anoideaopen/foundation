package mocks

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/msp"
)

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
