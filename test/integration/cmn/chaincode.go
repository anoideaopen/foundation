package cmn

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"path/filepath"

	"github.com/hyperledger/fabric/integration/nwo"
)

const nameACL = "acl"

func DeployChaincodeACL(network *nwo.Network, components *nwo.Components, ctor, testDir string) {
	nwo.DeployChaincode(network, nameACL, network.Orderers[0], nwo.Chaincode{
		Name:            "acl",
		Version:         "0.0",
		Path:            components.Build("github.com/anoideaopen/acl"),
		Lang:            "binary",
		PackageFile:     filepath.Join(testDir, "acl.tar.gz"),
		Ctor:            ctor,
		SignaturePolicy: `AND ('Org1MSP.member','Org2MSP.member')`,
		Sequence:        "1",
		InitRequired:    true,
		Label:           "my_prebuilt_chaincode",
	})
}

func NewSecrets(validators int) ([]ed25519.PrivateKey, error) {
	secrets := make([]ed25519.PrivateKey, 0, validators)
	for i := 0; i < validators; i++ {
		_, secret, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}

		secrets = append(secrets, secret)
	}

	return secrets, nil
}

func DeployChaincodeFoundation(
	network *nwo.Network,
	channel string,
	components *nwo.Components,
	path string,
	ctor string,
	testDir string,
) {
	nwo.DeployChaincode(network, channel, network.Orderers[0], nwo.Chaincode{
		Name:            channel,
		Version:         "0.0",
		Path:            components.Build(path),
		Lang:            "binary",
		PackageFile:     filepath.Join(testDir, fmt.Sprintf("%s.tar.gz", channel)),
		Ctor:            ctor,
		SignaturePolicy: `AND ('Org1MSP.member','Org2MSP.member')`,
		Sequence:        "1",
		InitRequired:    true,
		Label:           "my_prebuilt_chaincode",
	})
}
