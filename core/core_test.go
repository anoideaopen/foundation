package core

import (
	"os"
	"reflect"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/stretchr/testify/require"
)

func TestInvokeWithPanic(t *testing.T) {
	cc := ChainCode{}

	rsp := cc.Invoke(nil)
	require.Equal(t, int32(shim.ERROR), rsp.GetStatus())
	require.Equal(t, "panic invoke", rsp.GetMessage())
}

func TestWithTLS(t *testing.T) {
	expectedTLS := &TLS{
		Key:           []byte("test-key"),
		Cert:          []byte("test-cert"),
		ClientCACerts: []byte("test-ca"),
	}

	// Create a ChaincodeOption using the WithTLS function.
	option := WithTLS(expectedTLS)

	// Apply the ChaincodeOption to an empty chaincodeOptions instance.
	opts := &chaincodeOptions{}
	err := option(opts)
	if err != nil {
		t.Errorf("WithTLS failed with error: %v", err)
	}

	// Verify that the options were set correctly.
	if !reflect.DeepEqual(opts.TLS, expectedTLS) {
		t.Errorf("WithTLS did not set the expected TLS values")
	}
}

func TestWithTLSFromFiles(t *testing.T) {
	// Set up temporary files that will act as our TLS files.
	tempKeyFile, _ := os.CreateTemp("", "key.*.pem")
	tempCertFile, _ := os.CreateTemp("", "cert.*.pem")
	tempCAFile, _ := os.CreateTemp("", "ca.*.pem")

	defer func() {
		_ = os.Remove(tempKeyFile.Name())
		_ = os.Remove(tempCertFile.Name())
		_ = os.Remove(tempCAFile.Name())
	}()

	// Write dummy data to the temp files.
	_, _ = tempKeyFile.Write([]byte("key-data"))
	_, _ = tempCertFile.Write([]byte("cert-data"))
	_, _ = tempCAFile.Write([]byte("ca-data"))

	// Ensure files are written before we attempt to read.
	_ = tempKeyFile.Close()
	_ = tempCertFile.Close()
	_ = tempCAFile.Close()

	// Create a ChaincodeOption using the WithTLSFromFiles function.
	option, err := WithTLSFromFiles(tempKeyFile.Name(), tempCertFile.Name(), tempCAFile.Name())
	if err != nil {
		t.Fatalf("WithTLSFromFiles failed to create option: %v", err)
	}

	// Apply the ChaincodeOption to an empty chaincodeOptions instance.
	opts := &chaincodeOptions{}
	err = option(opts)
	if err != nil {
		t.Errorf("WithTLSFromFiles option application failed: %v", err)
	}

	// Verify that the options were set correctly.
	expectedTLS := &TLS{
		Key:           []byte("key-data"),
		Cert:          []byte("cert-data"),
		ClientCACerts: []byte("ca-data"),
	}
	if !reflect.DeepEqual(opts.TLS, expectedTLS) {
		t.Errorf("WithTLSFromFiles did not set the expected TLS values")
	}
}
