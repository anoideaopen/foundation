package hlfcreator

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"testing"

	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/stretchr/testify/require"
)

const userCert = `MIICSjCCAfGgAwIBAgIRAKeZTS2c/qkXBN0Vkh+0WYQwCgYIKoZIzj0EAwIwgYcx
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

const adminCert = `MIICSDCCAe6gAwIBAgIQAJwYy5PJAYSC1i0UgVN5bjAKBggqhkjOPQQDAjCBhzEL
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

const mspID = "mspID"

func Test_ValidateAdminCreator(t *testing.T) {
	type args struct {
		creator []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "nil creator", args: args{creator: nil}, wantErr: true},
		{name: "empty creator", args: args{creator: []byte{}}, wantErr: true},
		{name: "wrong creator", args: args{creator: []byte{12}}, wantErr: true},
		{name: "admin creator", args: args{creator: BuildCreator(t, mspID, adminCert)}, wantErr: false},
		{name: "client creator", args: args{creator: BuildCreator(t, mspID, userCert)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateAdminCreator(tt.args.creator); (err != nil) != tt.wantErr {
				t.Errorf("ValidateAdminCreator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_CreatorSKIAndHashedCert(t *testing.T) {
	type args struct {
		creator []byte
	}
	tests := []struct {
		name           string
		args           args
		wantCreatorSKI string
		wantHashedCert string
		wantErr        bool
	}{
		{
			name:    "nil creator",
			args:    args{creator: nil},
			wantErr: true,
		},
		{
			name:    "empty creator",
			args:    args{creator: []byte{}},
			wantErr: true,
		},
		{
			name:    "wrong creator",
			args:    args{creator: []byte{12}},
			wantErr: true,
		},
		{
			name:           "admin creator - mspID",
			args:           args{creator: BuildCreator(t, mspID, adminCert)},
			wantCreatorSKI: "dc752d6afb51c33327b7873fdb08adb91de15ee7c88f4f9949445aeeb8ea4e99",
			wantErr:        false,
		},
		{
			name:           "client creator - mspID",
			args:           args{creator: BuildCreator(t, mspID, userCert)},
			wantCreatorSKI: "7c77a240cf5ae2bdce217f352c93d90279db1f1f32196f90f5c36f633683bae3",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCreatorSKI, _, err := CreatorSKIAndHashedCert(tt.args.creator)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatorSKIAndHashedCert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				require.Equal(t, hex.EncodeToString(gotCreatorSKI[:]), tt.wantCreatorSKI)
			}
		})
	}
}

func BuildCreator(t *testing.T, creatorMSP string, creatorCert string) []byte {
	cert, err := base64.StdEncoding.DecodeString(creatorCert)
	require.NoError(t, err)
	pemblock := &pem.Block{Type: "CERTIFICATE", Bytes: cert}
	pemBytes := pem.EncodeToMemory(pemblock)
	require.NotNil(t, pemblock)

	creator := &msp.SerializedIdentity{Mspid: creatorMSP, IdBytes: pemBytes}
	marshaledIdentity, err := pb.Marshal(creator)
	require.NoError(t, err)
	return marshaledIdentity
}

func TestValidateSKI(t *testing.T) {
	var (
		one = [32]byte{1}
		two = [32]byte{2}
	)

	type args struct {
		sourceSKI          []byte
		expectedSKI        [32]byte
		expectedHashedCert [32]byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"source ski is nil or empty",
			args{
				sourceSKI:          nil,
				expectedSKI:        [32]byte{},
				expectedHashedCert: [32]byte{},
			},
			true,
		},
		{
			"source ski is nil or empty",
			args{
				sourceSKI:          []byte{},
				expectedSKI:        [32]byte{},
				expectedHashedCert: [32]byte{},
			},
			true,
		},
		{
			"source ski eq to expected ski and expected hashed cert",
			args{
				sourceSKI:          one[:],
				expectedSKI:        one,
				expectedHashedCert: one,
			},
			false,
		},
		{
			"source ski eq to expected ski but not eq expected hashed cert",
			args{
				sourceSKI:          one[:],
				expectedSKI:        one,
				expectedHashedCert: two,
			},
			false,
		},
		{
			"source ski eq to expected hashed cert but not eq expected ski",
			args{
				sourceSKI:          one[:],
				expectedSKI:        two,
				expectedHashedCert: one,
			},
			false,
		},
		{
			"source ski no eq to expected ski and not eq hashed cert",
			args{
				sourceSKI:          two[:],
				expectedSKI:        one,
				expectedHashedCert: one,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateSKI(tt.args.sourceSKI, tt.args.expectedSKI, tt.args.expectedHashedCert); (err != nil) != tt.wantErr {
				t.Errorf("ValidateSKI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
