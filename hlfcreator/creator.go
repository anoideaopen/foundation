package hlfcreator

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/sha3"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-protos-go-apiv2/msp"
	pb "google.golang.org/protobuf/proto"
)

const (
	// adminOU is the required OrganizationalUnit in the x509 certificate for Hyperledger admin.
	adminOU = "admin"
)

var ErrDecodeSerializedIdentity = errors.New("failed to validate block after decode pem 'SerializedIdentity.IdBytes', block can't be nil or empty")

// ValidateAdminCreator checks if the creator of the transaction is an admin.
func ValidateAdminCreator(creator []byte) error {
	if len(creator) == 0 {
		return errors.New("creator is nil or empty")
	}

	var identity msp.SerializedIdentity
	if err := pb.Unmarshal(creator, &identity); err != nil {
		return fmt.Errorf("failed to unmarshal SerializedIdentity %v: %w", creator, err)
	}

	b, _ := pem.Decode(identity.GetIdBytes())
	if b == nil || len(b.Bytes) == 0 {
		return ErrDecodeSerializedIdentity
	}
	parsed, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse x509 certificate: %w", err)
	}
	ouIsOk := false
	for _, ou := range parsed.Subject.OrganizationalUnit {
		if strings.ToLower(ou) == adminOU {
			ouIsOk = true
		}
	}
	if !ouIsOk {
		return fmt.Errorf("incorrect sender's OU, expected '%s' but found '%s'",
			adminOU,
			strings.Join(parsed.Subject.OrganizationalUnit, ","),
		)
	}

	return nil
}

func CreatorSKIAndHashedCert(creator []byte) (creatorSKI [32]byte, hashedCert [32]byte, err error) {
	if len(creator) == 0 {
		return creatorSKI, hashedCert, errors.New("creator is nil or empty")
	}

	var identity msp.SerializedIdentity
	if err = pb.Unmarshal(creator, &identity); err != nil {
		return creatorSKI, hashedCert, err
	}

	b, _ := pem.Decode(identity.GetIdBytes())
	if b == nil || len(b.Bytes) == 0 {
		return creatorSKI, hashedCert, ErrDecodeSerializedIdentity
	}
	parsed, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return creatorSKI, hashedCert, err
	}

	pk, ok := parsed.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return creatorSKI, hashedCert, errors.New("public key type requireion failed")
	}

	ecdhPk, err := pk.ECDH()
	if err != nil {
		return creatorSKI, hashedCert, fmt.Errorf("public key transition failed: %w", err)
	}
	creatorSKI = sha256.Sum256(ecdhPk.Bytes())
	hashedCert = sha3.Sum256(creator)

	return creatorSKI, hashedCert, nil
}

func ValidateSKI(sourceSKI []byte, expectedSKI [32]byte, expectedHashedCert [32]byte) error {
	if len(sourceSKI) == 0 {
		return errors.New("source ski is nil or empty")
	}

	if !bytes.Equal(expectedHashedCert[:], sourceSKI) &&
		!bytes.Equal(expectedSKI[:], sourceSKI) {
		sourceSKISum256 := sha256.Sum256(sourceSKI)
		expectedSKISum256 := sha256.Sum256(expectedSKI[:])
		expectedHashedCertSum256 := sha256.Sum256(expectedHashedCert[:])
		return fmt.Errorf("ski is not equal ski and hashed cert,"+
			" sourceSKI (hex sum256) %s, expectedSKI (hex sum256) %s, expectedHashedCert (hex sum256) %s",
			hex.EncodeToString(sourceSKISum256[:]),
			hex.EncodeToString(expectedSKISum256[:]),
			hex.EncodeToString(expectedHashedCertSum256[:]),
		)
	}

	return nil
}
