package cmn

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
)

// NetworkFoundation holds information about a fabric network.
type NetworkFoundation struct {
	*nwo.Network
	Robot     *Robot
	Templates *TemplatesFound
	Channels  []string

	mutex      sync.Locker
	colorIndex uint
}

func New(network *nwo.Network, channels []string) *NetworkFoundation {
	n := &NetworkFoundation{
		Network: network,
		Templates: &TemplatesFound{
			Templates: network.Templates,
		},
		Channels: channels,
		Robot:    &Robot{Ports: nwo.Ports{}},
		mutex:    &sync.Mutex{},
	}
	for _, portName := range RobotPortNames() {
		n.Robot.Ports[portName] = n.ReservePort()
	}

	return n
}

// Robot defines an orderer instance and its owning organization.
type Robot struct {
	Ports nwo.Ports `yaml:"ports,omitempty"`
}

func (n *NetworkFoundation) GenerateConfigTree() {
	n.Network.GenerateConfigTree()
	peer := n.Peer("Org1", "peer0")
	n.GenerateConnection(peer, "User1")
	n.GenerateConnection(peer, "User2")
	n.GenerateRobotConfig("User2")
}

// GenerateConnection creates the `connection.yaml` configuration file
// provided to profile `connection` for client. The path to the generated
// file can be obtained from ConnectionPath.
func (n *NetworkFoundation) GenerateConnection(p *nwo.Peer, u string) {
	config, err := os.Create(n.ConnectionPath(u))
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := config.Close()
		Expect(err).NotTo(HaveOccurred())
	}()

	t, err := template.New("connection").Funcs(template.FuncMap{
		"Peer": func() *nwo.Peer { return p },
		"User": func() string { return u },
	}).Parse(n.Templates.ConnectionTemplate())
	Expect(err).NotTo(HaveOccurred())

	pw := gexec.NewPrefixedWriter(fmt.Sprintf("[%s#connection.yaml] ", u), ginkgo.GinkgoWriter)
	err = t.Execute(io.MultiWriter(config, pw), n)
	Expect(err).NotTo(HaveOccurred())
}

// GenerateRobotConfig creates the `robot.yaml` configuration file
// provided to config for robot. The path to the generated
// file can be obtained from RobotPath.
func (n *NetworkFoundation) GenerateRobotConfig(u string) {
	config, err := os.Create(n.RobotPath())
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := config.Close()
		Expect(err).NotTo(HaveOccurred())
	}()

	t, err := template.New("robot").Funcs(template.FuncMap{
		"User": func() string { return u },
	}).Parse(n.Templates.RobotTemplate())
	Expect(err).NotTo(HaveOccurred())

	pw := gexec.NewPrefixedWriter("[robot.yaml] ", ginkgo.GinkgoWriter)
	err = t.Execute(io.MultiWriter(config, pw), n)
	Expect(err).NotTo(HaveOccurred())
}

// ConnectionPath returns the path to the generated connection profile file.
func (n *NetworkFoundation) ConnectionPath(user string) string {
	return filepath.Join(n.RootDir, user+"_connection.yaml")
}

// RobotPath returns the path to the generated robot profile file.
func (n *NetworkFoundation) RobotPath() string {
	return filepath.Join(n.RootDir, "robot.yaml")
}

// PeerUserKeyFound returns the path to the private key for the specified user in
// the peer organization.
func (n *NetworkFoundation) PeerUserKeyFound(p *nwo.Peer, user string) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	keystore := filepath.Join(
		n.PeerUserMSPDir(p, user),
		"keystore",
	)

	// file names are the SKI and non-deterministic
	keys, err := os.ReadDir(keystore)
	if err != nil {
		return filepath.Join(keystore, "priv_sk")
	}

	Expect(keys).To(HaveLen(1))

	return filepath.Join(keystore, keys[0].Name())
}

// RobotPortNames  returns the list of ports that need to be reserved for a robot.
func RobotPortNames() []nwo.PortName {
	return []nwo.PortName{nwo.ListenPort}
}

// RobotPort returns the named port reserved for the Robot instance.
func (n *NetworkFoundation) RobotPort(portName nwo.PortName) uint16 {
	peerPorts := n.Robot.Ports
	Expect(peerPorts).NotTo(BeNil())
	return peerPorts[portName]
}

// RobotRunner returns an ifrit.Runner for the specified robot. The runner can be
// used to start and manage a robot process.
func (n *NetworkFoundation) RobotRunner(env ...string) *ginkgomon.Runner {
	cmd := exec.Command(n.Components.Build(RobotModulePath()), "-c", n.RobotPath())
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)

	return ginkgomon.New(ginkgomon.Config{
		AnsiColorCode:     n.nextColor(),
		Name:              "robot",
		Command:           cmd,
		StartCheck:        `Robot started, time -`,
		StartCheckTimeout: 15 * time.Second,
	})
}

func (n *NetworkFoundation) nextColor() string {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	cInd := n.colorIndex + 7
	color := cInd%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	n.colorIndex++
	return fmt.Sprintf("%dm", color)
}

func RobotModulePath() string {
	return "github.com/anoideaopen/robot"
}

// PeerTLSCACert returns the path to the local tlsca cert for the peer.
func (n *NetworkFoundation) PeerTLSCACert(p *nwo.Peer) string {
	dirName := filepath.Join(n.PeerLocalMSPDir(p), "tlscacerts")
	fileName := fmt.Sprintf("tlsca.%s-cert.pem", n.Organization(p.Organization).Domain)
	return filepath.Join(dirName, fileName)
}

// OrdererTLSCACert returns the path to the local tlsca cert for the Orderer.
func (n *NetworkFoundation) OrdererTLSCACert(o *nwo.Orderer) string {
	dirName := filepath.Join(n.OrdererLocalMSPDir(o), "tlscacerts")
	fileName := fmt.Sprintf("tlsca.%s-cert.pem", n.Organization(o.Organization).Domain)
	return filepath.Join(dirName, fileName)
}
