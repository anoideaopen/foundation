package client

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/anoideaopen/acl/cc"
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

// AddUser adds user to ACL channel
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

// AddAdminToACL adds admin user to ACL
func (ts *testSuite) AddAdminToACL() {
	ts.AddUser(ts.admin)
}

// AddFeeSetterToACL adds fee setter user to ACL
func (ts *testSuite) AddFeeSetterToACL() {
	ts.AddUser(ts.feeSetter)
}

// AddFeeAddressSetterToACL adds fee address setter to ACL
func (ts *testSuite) AddFeeAddressSetterToACL() {
	ts.AddUser(ts.feeAddressSetter)
}

// AddUserMultisigned adds multisigned user to ACL channel
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

// CheckUser checks if user was added to ACL channel
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

// CheckUserMultisigned checks if multisigned user was added to ACL channel
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

// AddRights adds right for defined user with specified role and operation to ACL channel
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

// RemoveRights removes right for defined user with specified role and operation to ACL channel
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
	nc := NewNonceByTime().Get()
	// ToDo - Why are we signing arguments that differs we are sending?
	ctorArgsToSign := []string{common.FnChangeMultisigPublicKey, multisignedUser.AddressBase58Check, oldPubKeyBase58, multisignedUser.PublicKey(), reason, reasonID, nc}
	validatorMultisignedUser := &UserFoundationMultisigned{
		UserID: "multisigned validators",
		Users:  validators,
	}

	pKeys, sMsgsByte, err := validatorMultisignedUser.Sign(ctorArgsToSign...)
	Expect(err).NotTo(HaveOccurred())

	var sMsgsStr []string
	ctorArgs := []string{common.FnChangeMultisigPublicKey, multisignedUser.AddressBase58Check, oldPubKeyBase58, newPubKeyBase58, reason, reasonID, nc}
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

// AddToBlackList adds user to a black list
func (ts *testSuite) AddToBlackList(user *UserFoundation) {
	ts.addToList(cc.BlackList, user)
}

// AddToGrayList adds user to a gray list
func (ts *testSuite) AddToGrayList(user *UserFoundation) {
	ts.addToList(cc.GrayList, user)
}

func (ts *testSuite) addToList(listType cc.ListType, user *UserFoundation) {
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor: cmn.CtorFromSlice([]string{
			common.FnAddToList,
			user.AddressBase58Check,
			listType.String(),
		}),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckUserInList(listType, user)
}

// DelFromBlackList adds user to a black list
func (ts *testSuite) DelFromBlackList(user *UserFoundation) {
	ts.delFromList(cc.BlackList, user)
}

// DelFromGrayList adds user to a gray list
func (ts *testSuite) DelFromGrayList(user *UserFoundation) {
	ts.delFromList(cc.GrayList, user)
}

func (ts *testSuite) delFromList(
	listType cc.ListType,
	user *UserFoundation,
) {
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor: cmn.CtorFromSlice([]string{
			common.FnDelFromList,
			user.AddressBase58Check,
			listType.String(),
		}),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckUserNotInList(listType, user)
}

// CheckUserInList - checks if user in gray or black list
func (ts *testSuite) CheckUserInList(listType cc.ListType, user *UserFoundation) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{common.FnCheckKeys, user.PublicKeyBase58}),
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

		account := resp.GetAccount()
		if !((account.GetBlackListed() && listType == cc.BlackList) || (account.GetGrayListed() && listType == cc.GrayList)) {
			return fmt.Sprintf("Error: expected %s to be added to %s", user.AddressBase58Check, listType.String())
		}

		return ""
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

// CheckUserNotInList - checks if user in gray or black list
func (ts *testSuite) CheckUserNotInList(listType cc.ListType, user *UserFoundation) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{common.FnCheckKeys, user.PublicKeyBase58}),
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

		account := resp.GetAccount()
		if !((!account.GetBlackListed() && listType == cc.BlackList) || (!account.GetGrayListed() && listType == cc.GrayList)) {
			return fmt.Sprintf("Error: expected %s to be deleted from %s", user.AddressBase58Check, listType.String())
		}

		return ""
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

// ChangePublicKey - changes user public key by validators
func (ts *testSuite) ChangePublicKey(
	user *UserFoundation,
	newPubKeyBase58 string,
	reason string,
	reasonID string,
	validators ...*UserFoundation,
) {
	ctorArgs := []string{common.FnChangePublicKey, user.AddressBase58Check, reason, reasonID, newPubKeyBase58, NewNonceByTime().Get()}
	validatorMultisignedUser := &UserFoundationMultisigned{
		UserID: "multisigned validators",
		Users:  validators,
	}

	pKeys, sMsgsByte, err := validatorMultisignedUser.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	var sMsgsStr []string
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

	ts.CheckUserChangedKey(newPubKeyBase58, user.AddressBase58Check)
}

// ChangePublicKeyBase58signed - changes user public key by validators with base58 signatures
func (ts *testSuite) ChangePublicKeyBase58signed(
	user *UserFoundation,
	requestID string,
	chaincodeName string,
	channelID string,
	newPubKeyBase58 string,
	reason string,
	reasonID string,
	validators ...*UserFoundation,
) {
	ctorArgs := []string{common.FnChangePublicKeyWithBase58Signature, requestID, chaincodeName, channelID, user.AddressBase58Check, reason, reasonID, newPubKeyBase58, NewNonceByTime().Get()}
	validatorMultisignedUser := &UserFoundationMultisigned{
		UserID: "multisigned validators",
		Users:  validators,
	}

	pKeys, sMsgsByte, err := validatorMultisignedUser.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	var sMsgsStr []string
	for _, sMsgByte := range sMsgsByte {
		sMsgsStr = append(sMsgsStr, base58.Encode(sMsgByte))
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

	ts.CheckUserChangedKey(newPubKeyBase58, user.AddressBase58Check)
}

// CheckUserChangedKey checks if user changed key
func (ts *testSuite) CheckUserChangedKey(newPublicKeyBase58Check, oldAddressBase58Check string) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{"checkKeys", newPublicKeyBase58Check}),
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
		if addr != oldAddressBase58Check {
			return fmt.Sprintf("Error: expected %s, received %s", oldAddressBase58Check, addr)
		}

		return ""
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

// CheckAccountInfo checks account info
func (ts *testSuite) CheckAccountInfo(
	user *UserFoundation,
	kycHash string,
	isGrayListed,
	isBlackListed bool,
) {
	Eventually(func() string {
		sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      cmn.CtorFromSlice([]string{common.FnGetAccInfoFn, user.AddressBase58Check}),
		})
		Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit())
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		resp := &pb.AccountInfo{}
		err = json.Unmarshal(out, resp)
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		if resp.GetKycHash() != kycHash {
			return fmt.Sprintf("kyc check error: expected %s, received %s", kycHash, resp.GetKycHash())
		}

		if resp.GetGrayListed() != isGrayListed {
			return fmt.Sprintf("gray list check error error: expected %t, received %t", isGrayListed, resp.GetGrayListed())
		}

		if resp.GetBlackListed() != isBlackListed {
			return fmt.Sprintf("black list check error: expected %t, received %t", isBlackListed, resp.GetBlackListed())
		}

		return ""
	}, ts.network.EventuallyTimeout, time.Second).Should(BeEmpty())
}

// SetAccountInfo sets account info
func (ts *testSuite) SetAccountInfo(
	user *UserFoundation,
	kycHash string,
	isGrayListed,
	isBlackListed bool,
) {
	sess, err := ts.network.PeerUserSession(ts.peer, ts.mainUserName, commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   ts.network.OrdererAddress(ts.orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor: cmn.CtorFromSlice([]string{
			"setAccountInfo",
			user.AddressBase58Check,
			kycHash,
			strconv.FormatBool(isGrayListed),
			strconv.FormatBool(isBlackListed),
		}),
		PeerAddresses: []string{
			ts.network.PeerAddress(ts.network.Peer(ts.org1Name, ts.peer.Name), nwo.ListenPort),
			ts.network.PeerAddress(ts.network.Peer(ts.org2Name, ts.peer.Name), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, ts.network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	ts.CheckAccountInfo(user, kycHash, isGrayListed, isBlackListed)
}

// SetKYC sets kyc hash
func (ts *testSuite) SetKYC(
	user *UserFoundation,
	kycHash string,
	validators ...*UserFoundation,
) {
	ctorArgs := []string{common.FnSetKYC, user.AddressBase58Check, kycHash, NewNonceByTime().Get()}
	validatorMultisignedUser := &UserFoundationMultisigned{
		UserID: "multisigned validators",
		Users:  validators,
	}

	pKeys, sMsgsByte, err := validatorMultisignedUser.Sign(ctorArgs...)
	Expect(err).NotTo(HaveOccurred())

	var sMsgsStr []string
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

	ts.CheckAccountInfo(user, kycHash, false, false)
}
