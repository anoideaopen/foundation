package mocks

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"testing"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
)

const DefaultCert = `MIICSjCCAfGgAwIBAgIRAKeZTS2c/qkXBN0Vkh+0WYQwCgYIKoZIzj0EAwIwgYcx
CzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4g
RnJhbmNpc2NvMSMwIQYDVQQKExphdG9teXplLnVhdC5kbHQuYXRvbXl6ZS5jaDEm
MCQGA1UEAxMdY2EuYXRvbXl6ZS51YXQuZGx0LmF0b215emUuY2gwHhcNMjAxMDEz
MDg1NjAwWhcNMzAxMDExMDg1NjAwWjB3MQswCQYDVQQGEwJVUzETMBEGA1UECBMK
Q2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZyYW5jaXNjbzEPMA0GA1UECxMGY2xp
ZW50MSowKAYDVQQDDCFVc2VyMTBAYXRvbXl6ZS51YXQuZGx0LmF0b215emUuY2gw
WTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAR3V6z/nVq66HBDxFFN3/3rUaJLvHgW
FzoKaA/qZQyV919gdKr82LDy8N2kAYpAcP7dMyxMmmGOPbo53locYWIyo00wSzAO
BgNVHQ8BAf8EBAMCB4AwDAYDVR0TAQH/BAIwADArBgNVHSMEJDAigCBSv0ueZaB3
qWu/AwOtbOjaLd68woAqAklfKKhfu10K+DAKBggqhkjOPQQDAgNHADBEAiBFB6RK
O7huI84Dy3fXeA324ezuqpJJkfQOJWkbHjL+pQIgFKIqBJrDl37uXNd3eRGJTL+o
21ZL8pGXH8h0nHjOF9M=`

const AdminCert = `MIICSDCCAe6gAwIBAgIQAJwYy5PJAYSC1i0UgVN5bjAKBggqhkjOPQQDAjCBhzEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xIzAhBgNVBAoTGmF0b215emUudWF0LmRsdC5hdG9teXplLmNoMSYw
JAYDVQQDEx1jYS5hdG9teXplLnVhdC5kbHQuYXRvbXl6ZS5jaDAeFw0yMDEwMTMw
ODU2MDBaFw0zMDEwMTEwODU2MDBaMHUxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMQ4wDAYDVQQLEwVhZG1p
bjEpMCcGA1UEAwwgQWRtaW5AYXRvbXl6ZS51YXQuZGx0LmF0b215emUuY2gwWTAT
BgcqhkjOPQIBBggqhkjOPQMBBwNCAAQGQX9IhgjCtd3mYZ9DUszmUgvubepVMPD5
FlwjCglB2SiWuE2rT/T5tHJsU/Y9ZXFtOOpy/g9tQ/0wxDWwpkbro00wSzAOBgNV
HQ8BAf8EBAMCB4AwDAYDVR0TAQH/BAIwADArBgNVHSMEJDAigCBSv0ueZaB3qWu/
AwOtbOjaLd68woAqAklfKKhfu10K+DAKBggqhkjOPQQDAgNIADBFAiEAoKRQLe4U
FfAAwQs3RCWpevOPq+J8T4KEsYvswKjzfJYCIAs2kOmN/AsVUF63unXJY0k9ktfD
fAaqNRaboY1Yg1iQ`

const TestCreatorMSP = "platformMSP"

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

// NewMockStub returns new mock stub
func NewMockStub(t *testing.T) *ChaincodeStub {
	mockStub := new(ChaincodeStub)
	txID := [16]byte(uuid.New())
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))
	mockStub.GetSignedProposalReturns(&peer.SignedProposal{}, nil)

	err := SetCreatorCert(mockStub, TestCreatorMSP, AdminCert)
	require.NoError(t, err)

	return mockStub
}
