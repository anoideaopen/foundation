package client

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/anoideaopen/acl/tests/common"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"google.golang.org/protobuf/encoding/protojson"
)

// Deprecated: need to remove after migrating to testsuite
// AddUser adds user to ACL channel
func AddUser(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	user *UserFoundation,
) {
	sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor: cmn.CtorFromSlice(
			[]string{
				"addUserWithPublicKeyType",
				user.PublicKeyBase58,
				"test",
				user.UserID,
				"true",
				user.KeyType.String(),
			},
		),
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	CheckUser(network, peer, user)
}

// Deprecated: need to remove after migrating to testsuite
// AddUserMultisigned adds multisigned user to ACL channel
func AddUserMultisigned(network *nwo.Network, peer *nwo.Peer, orderer *nwo.Orderer, n int, user *UserFoundationMultisigned) {
	ctorArgs := []string{common.FnAddMultisig, strconv.Itoa(n), NewNonceByTime().Get()}
	publicKeys, sMsgsByte, err := user.Sign(ctorArgs...)
	var sMsgsStr []string
	for _, sMsgByte := range sMsgsByte {
		sMsgsStr = append(sMsgsStr, hex.EncodeToString(sMsgByte))
	}
	ctorArgs = append(append(ctorArgs, publicKeys...), sMsgsStr...)
	sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice(ctorArgs),
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	CheckUserMultisigned(network, peer, user)
}

// Deprecated: need to remove after migrating to testsuite
// CheckUser checks if user was added to ACL channel
func CheckUser(network *nwo.Network, peer *nwo.Peer, user *UserFoundation) {
	Eventually(func() string {
		sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{"checkKeys", user.PublicKeyBase58}),
		})
		Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit())
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		resp := &pb.AclResponse{}
		err = proto.Unmarshal(out, resp)
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		addr := base58.CheckEncode(resp.GetAddress().GetAddress().GetAddress()[1:], resp.GetAddress().GetAddress().GetAddress()[0])
		if addr != user.AddressBase58Check {
			return fmt.Sprintf("Error: expected %s, received %s", user.AddressBase58Check, addr)
		}

		return ""
	}, network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

// Deprecated: need to remove after migrating to testsuite
// CheckUserMultisigned checks if multisigned user was added to ACL channel
func CheckUserMultisigned(network *nwo.Network, peer *nwo.Peer, user *UserFoundationMultisigned) {
	Eventually(func() string {
		sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{common.FnCheckKeys, user.PublicKey()}),
		})
		Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit())
		Expect(sess.ExitCode()).To(Equal(0))
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		resp := &pb.AclResponse{}
		err = proto.Unmarshal(out, resp)
		Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		addressBytes := resp.GetAddress().GetAddress().GetAddress()
		addr := base58.CheckEncode(addressBytes[1:], addressBytes[0])
		if addr != user.AddressBase58Check {
			return fmt.Sprintf("Error: expected %s, received %s", user.AddressBase58Check, addr)
		}

		return ""
	}, network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

// Deprecated: need to remove after migrating to testsuite
// ChangeMultisigPublicKey changes public key for multisigned user by validators
func ChangeMultisigPublicKey(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	multisignedUser *UserFoundationMultisigned,
	oldPubKeyBase58 string,
	newPubKeyBase58 string,
	reason string,
	reasonID string,
	validators ...*UserFoundation,
) {
	nonce := NewNonceByTime().Get()
	// ToDo - Why are we signing arguments that differs we are sending?
	ctorArgsToSign := []string{common.FnChangeMultisigPublicKey, multisignedUser.AddressBase58Check, oldPubKeyBase58, multisignedUser.PublicKey(), reason, reasonID, nonce}
	validatorMultisignedUser := &UserFoundationMultisigned{
		UserID: "multisigned validators",
		Users:  validators,
	}

	pKeys, sMsgsByte, err := validatorMultisignedUser.Sign(ctorArgsToSign...)
	Expect(err).NotTo(HaveOccurred())

	var sMsgsStr []string
	ctorArgs := []string{common.FnChangeMultisigPublicKey, multisignedUser.AddressBase58Check, oldPubKeyBase58, newPubKeyBase58, reason, reasonID, nonce}
	for _, sMsgByte := range sMsgsByte {
		sMsgsStr = append(sMsgsStr, hex.EncodeToString(sMsgByte))
	}

	ctorArgs = append(append(ctorArgs, pKeys...), sMsgsStr...)

	sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice(ctorArgs),
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	CheckUserMultisigned(network, peer, multisignedUser)
}

// Deprecated: need to remove after migrating to testsuite
// AddRights adds right for defined user with specified role and operation to ACL channel
func AddRights(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	channel string,
	cc string,
	role string,
	operation string,
	user *UserFoundation,
) {
	args := []string{"addRights", channel, cc, role, operation, user.AddressBase58Check}
	sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice(args),
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	CheckRights(network, peer, channel, cc, role, operation, user, true)
}

// Deprecated: need to remove after migrating to testsuite
// RemoveRights removes right for defined user with specified role and operation to ACL channel
func RemoveRights(
	network *nwo.Network,
	peer *nwo.Peer,
	orderer *nwo.Orderer,
	channel string,
	cc string,
	role string,
	operation string,
	user *UserFoundation,
) {
	sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice([]string{"removeRights", channel, cc, role, operation, user.AddressBase58Check}),
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	CheckRights(network, peer, channel, cc, role, operation, user, false)
}

// Deprecated: need to remove after migrating to testsuite
func CheckRights(
	network *nwo.Network,
	peer *nwo.Peer,
	channel string,
	cc string,
	role string,
	operation string,
	user *UserFoundation,
	result bool,
) {
	Eventually(func() string {
		sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{"getAccountOperationRightJSON", channel, cc, role, operation, user.AddressBase58Check}),
		})
		Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit())
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		haveRight := &pb.HaveRight{}
		err = protojson.Unmarshal(out, haveRight)
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		if haveRight.GetHaveRight() != result {
			return fmt.Sprintf("Error: expected %t, received %t", result, haveRight.GetHaveRight())
		}

		return ""
	}, network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

func (ts *testSuite) AddUser(user *UserFoundation) {
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor: cmn.CtorFromSlice(
			[]string{
				"addUserWithPublicKeyType",
				user.PublicKeyBase58,
				"test",
				user.UserID,
				"true",
				user.KeyType.String(),
			},
		),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckUser(user)
}

func (ts *testSuite) AddAdminToACL() {
	ts.AddUser(ts.admin)
}

func (ts *testSuite) AddFeeSetterToACL() {
	ts.AddUser(ts.feeSetter)
}

func (ts *testSuite) AddFeeAddressSetterToACL() {
	ts.AddUser(ts.feeAddressSetter)
}

func (ts *testSuite) AddUserMultisigned(user *UserFoundationMultisigned) {
	ctorArgs := []string{common.FnAddMultisig, strconv.Itoa(len(user.Users)), NewNonceByTime().Get()}
	publicKeys, sMsgsByte, err := user.Sign(ctorArgs...)
	var sMsgsStr []string
	for _, sMsgByte := range sMsgsByte {
		sMsgsStr = append(sMsgsStr, hex.EncodeToString(sMsgByte))
	}
	ctorArgs = append(append(ctorArgs, publicKeys...), sMsgsStr...)
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice(ctorArgs),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckUserMultisigned(user)
}

func (ts *testSuite) CheckUser(user *UserFoundation) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{"checkKeys", user.PublicKeyBase58}),
		})
		Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit())
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		resp := &pb.AclResponse{}
		err = proto.Unmarshal(out, resp)
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		addr := base58.CheckEncode(resp.GetAddress().GetAddress().GetAddress()[1:], resp.GetAddress().GetAddress().GetAddress()[0])
		if addr != user.AddressBase58Check {
			return fmt.Sprintf("Error: expected %s, received %s", user.AddressBase58Check, addr)
		}

		return ""
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

func (ts *testSuite) CheckUserMultisigned(user *UserFoundationMultisigned) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{common.FnCheckKeys, user.PublicKey()}),
		})
		Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit())
		Expect(sess.ExitCode()).To(Equal(0))
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		resp := &pb.AclResponse{}
		err = proto.Unmarshal(out, resp)
		Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		addressBytes := resp.GetAddress().GetAddress().GetAddress()
		addr := base58.CheckEncode(addressBytes[1:], addressBytes[0])
		if addr != user.AddressBase58Check {
			return fmt.Sprintf("Error: expected %s, received %s", user.AddressBase58Check, addr)
		}

		return ""
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

func (ts *testSuite) AddRights(channelName, chaincodeName, role, operation string, user *UserFoundation) {
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice([]string{"addRights", channelName, chaincodeName, role, operation, user.AddressBase58Check}),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckRights(channelName, chaincodeName, role, operation, user, true)
}

func (ts *testSuite) RemoveRights(channelName, chaincodeName, role, operation string, user *UserFoundation) {
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice([]string{"removeRights", channelName, chaincodeName, role, operation, user.AddressBase58Check}),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckRights(channelName, chaincodeName, role, operation, user, false)
}

func (ts *testSuite) CheckRights(channelName, chaincodeName, role, operation string, user *UserFoundation, result bool) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{"getAccountOperationRightJSON", channelName, chaincodeName, role, operation, user.AddressBase58Check}),
		})
		Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit())
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		haveRight := &pb.HaveRight{}
		err = protojson.Unmarshal(out, haveRight)
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		if haveRight.GetHaveRight() != result {
			return fmt.Sprintf("Error: expected %t, received %t", result, haveRight.GetHaveRight())
		}

		return ""
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

// ChangeMultisigPublicKey changes public key for multisigned user by validators
func (ts *testSuite) ChangeMultisigPublicKey(
	multisignedUser *UserFoundationMultisigned,
	oldPubKeyBase58 string,
	newPubKeyBase58 string,
	reason string,
	reasonID string,
	validators ...*UserFoundation,
) {
	nonce := NewNonceByTime().Get()
	// ToDo - Why are we signing arguments that differs we are sending?
	ctorArgsToSign := []string{common.FnChangeMultisigPublicKey, multisignedUser.AddressBase58Check, oldPubKeyBase58, multisignedUser.PublicKey(), reason, reasonID, nonce}
	validatorMultisignedUser := &UserFoundationMultisigned{
		UserID: "multisigned validators",
		Users:  validators,
	}

	pKeys, sMsgsByte, err := validatorMultisignedUser.Sign(ctorArgsToSign...)
	Expect(err).NotTo(HaveOccurred())

	var sMsgsStr []string
	ctorArgs := []string{common.FnChangeMultisigPublicKey, multisignedUser.AddressBase58Check, oldPubKeyBase58, newPubKeyBase58, reason, reasonID, nonce}
	for _, sMsgByte := range sMsgsByte {
		sMsgsStr = append(sMsgsStr, hex.EncodeToString(sMsgByte))
	}

	ctorArgs = append(append(ctorArgs, pKeys...), sMsgsStr...)

	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      cmn.CtorFromSlice(ctorArgs),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckUserMultisigned(multisignedUser)
}
