package cmn

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"path/filepath"

	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

const nameACL = "acl"

func DeployChaincodeACL(network *nwo.Network, components *nwo.Components, ctor, testDir string) {
	nwo.DeployChaincode(network, nameACL, network.Orderers[0], nwo.Chaincode{
		Name:            nameACL,
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
	DeployChaincode(network, channel, network.Orderers[0], nwo.Chaincode{
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

// DeployChaincode is a helper that will install chaincode to all peers that
// are connected to the specified channel, approve the chaincode on one of the
// peers of each organization in the network, commit the chaincode definition
// on the channel using one of the peers, and wait for the chaincode commit to
// complete on all of the peers. It uses the _lifecycle implementation.
// NOTE V2_0 capabilities must be enabled for this functionality to work.
func DeployChaincode(n *nwo.Network, channel string, orderer *nwo.Orderer, chaincode nwo.Chaincode, peers ...*nwo.Peer) {
	if len(peers) == 0 {
		peers = n.PeersWithChannel(channel)
	}
	if len(peers) == 0 {
		return
	}

	nwo.PackageAndInstallChaincode(n, chaincode, peers...)

	// approve for each org
	nwo.ApproveChaincodeForMyOrg(n, channel, orderer, chaincode, peers...)

	// wait for checkcommitreadiness returns ready status
	nwo.CheckCommitReadinessUntilReady(n, channel, chaincode, n.PeerOrgs(), peers...)

	// after the chaincode definition has been correctly approved for each org,
	// demonstrate the capability to inspect the discrepancies in the chaincode definitions
	// by executing checkcommitreadiness with inspect flag,
	// with intentionally altered values for chaincode definition parameters
	nwo.InspectChaincodeDiscrepancies(n, channel, chaincode, n.PeerOrgs(), peers...)

	// commit definition
	nwo.CommitChaincode(n, channel, orderer, chaincode, peers[0], peers...)

	// init the chaincode, if required
	if chaincode.InitRequired {
		InitChaincode(n, channel, orderer, chaincode, peers...)
	}
}

func InitChaincode(n *nwo.Network, channel string, orderer *nwo.Orderer, chaincode nwo.Chaincode, peers ...*nwo.Peer) {
	// init using one peer per org
	initOrgs := map[string]bool{}
	var peerAddresses []string
	for _, p := range peers {
		if exists := initOrgs[p.Organization]; !exists {
			peerAddresses = append(peerAddresses, n.PeerAddress(p, nwo.ListenPort))
			initOrgs[p.Organization] = true
		}
	}

	sess, err := n.PeerAdminSession(peers[0], commands.ChaincodeInvoke{
		ChannelID:     channel,
		Orderer:       n.OrdererAddress(orderer, nwo.ListenPort),
		Name:          chaincode.Name,
		Ctor:          chaincode.Ctor,
		PeerAddresses: peerAddresses,
		WaitForEvent:  true,
		IsInit:        true,
		ClientAuth:    n.ClientAuthRequired,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	for i := 0; i < len(peerAddresses); i++ {
		Eventually(sess.Err, n.EventuallyTimeout).Should(gbytes.Say(`\Qcommitted with status (VALID)\E`))
	}
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))
}
