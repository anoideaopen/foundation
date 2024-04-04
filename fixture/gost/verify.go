package main

import (
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"github.com/ddulesov/gogost/gost3410"
	"github.com/ddulesov/gogost/gost34112012256"
)

type SubjectPublicKeyInfo struct {
	Algorithm        AlgorithmIdentifier
	SubjectPublicKey asn1.BitString
}

type AlgorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue
}

func main() {
	// Load the public key from the PEM file
	pubKeyPEM, err := os.ReadFile("public_key.pem.example")
	if err != nil {
		log.Fatalf("failed to read public key file: %v", err)
	}
	blockPub, _ := pem.Decode(pubKeyPEM)
	if blockPub == nil {
		log.Fatalf("failed to parse public key PEM")
	}

	// Extract the public key from the ASN.1 structure
	var publicKeyInfo SubjectPublicKeyInfo
	_, err = asn1.Unmarshal(blockPub.Bytes, &publicKeyInfo)
	if err != nil {
		log.Fatalf("failed to unmarshal ASN.1 public key: %v", err)
	}

	pubKeyBytes := publicKeyInfo.SubjectPublicKey.Bytes[2:]

	// Initialize the public key
	publicKey, err := gost3410.NewPublicKey(
		gost3410.CurveIdGostR34102001CryptoProAParamSet(),
		gost3410.Mode2001,
		pubKeyBytes,
	)
	if err != nil {
		log.Fatalf("failed to parse public key: %v", err)
	}

	// Read Base64 signature from a file
	signatureBase64, err := os.ReadFile("signature_base64.txt")
	if err != nil {
		log.Fatalf("failed to read signature file: %v", err)
	}
	signatureRaw, err := base64.StdEncoding.DecodeString(string(signatureBase64))
	if err != nil {
		log.Fatalf("failed to decode base64 signature: %v", err)
	}

	// Read the original message from the file
	message, err := os.ReadFile("data.txt")
	if err != nil {
		log.Fatalf("failed to read data file: %v", err)
	}

	// Calculating the message digest
	digest := gost34112012256.New()
	if _, err := digest.Write(message); err != nil {
		log.Fatalf("failed to hash message: %v", err)
	}
	hash := digest.Sum(nil)

	// Display message digest
	fmt.Println("Message digest:", base64.StdEncoding.EncodeToString(hash))

	// Checking the signature
	ok, err := publicKey.VerifyDigest(reverseBytes(hash), signatureRaw)
	if err != nil {
		log.Fatalf("failed to verify signature: %v", err)
	}

	if ok {
		fmt.Println("The signature is correct.")
	} else {
		fmt.Println("The signature is incorrect.")
	}
}

// reverseBytes returns a new byte slice that is the inverse of the passed slice.
func reverseBytes(in []byte) []byte {
	n := len(in)
	reversed := make([]byte, n)
	for i, b := range in {
		reversed[n-i-1] = b
	}
	return reversed
}
