package mock

import (
	"errors"

	"github.com/hyperledger/fabric-chaincode-go/shim"
)

const (
	chNameACL = "acl"
)

// Errors
const (
	ErrRightNotSet = "right is not set"

	ErrChannelNotSet   = "right is broken, channel is not set"
	ErrChaincodeNotSet = "right is broken, chaincode is not set"
	ErrRoleNotSet      = "right is broken, role is not set"
	ErrAddressNotSet   = "right is broken, address is not set"

	ErrNomineeAddressNotSet   = "right is broken, nominee address is not set"
	ErrPrincipalAddressNotSet = "right is broken, principal address is not set"
)

// Function names
const (
	// FnAddRights adds a right to the access matrix
	FnAddRights operation = "addRights"
	// FnRemoveRights removes a right from the access matrix
	FnRemoveRights operation = "removeRights"
	// FnAddAddressRightForNominee adds right to access matrix
	FnAddAddressRightForNominee operation = "addAddressRightForNominee"
	// FnRemoveAddressRightFromNominee adds right to access matrix
	FnRemoveAddressRightFromNominee operation = "removeAddressRightFromNominee"
)

// Right defines a right in the access matrix
type Right struct {
	Channel   string
	Chaincode string
	Role      string
	Operation string
	Address   string
}

// AddressRight defines address right for nominee in access matrix
type AddressRight struct {
	Channel          string
	Chaincode        string
	NomineeAddress   string
	PrincipalAddress string
}

// IsValid checks if the right is valid
func (r Right) IsValid() error {
	if len(r.Channel) == 0 {
		return errors.New(ErrChannelNotSet)
	}

	if len(r.Chaincode) == 0 {
		return errors.New(ErrChaincodeNotSet)
	}

	if len(r.Role) == 0 {
		return errors.New(ErrRoleNotSet)
	}

	if len(r.Address) == 0 {
		return errors.New(ErrAddressNotSet)
	}

	return nil
}

func (ar AddressRight) IsValid() error {
	if len(ar.Channel) == 0 {
		return errors.New(ErrChannelNotSet)
	}

	if len(ar.Chaincode) == 0 {
		return errors.New(ErrChaincodeNotSet)
	}

	if len(ar.NomineeAddress) == 0 {
		return errors.New(ErrNomineeAddressNotSet)
	}

	if len(ar.PrincipalAddress) == 0 {
		return errors.New(ErrPrincipalAddressNotSet)
	}

	return nil
}

type operation string

// Deprecated: use package ../mocks instead
// AddAccountRight adds a right to the access matrix
func (w *Wallet) AddAccountRight(right *Right) error {
	return w.modifyRight(FnAddRights, right)
}

// Deprecated: use package ../mocks instead
// RemoveAccountRight removes a right from the access matrix
func (w *Wallet) RemoveAccountRight(right *Right) error {
	return w.modifyRight(FnRemoveRights, right)
}

// Deprecated: use package ../mocks instead
func (w *Wallet) modifyRight(opFn operation, right *Right) error {
	if right == nil {
		return errors.New(ErrRightNotSet)
	}

	validationErr := right.IsValid()
	if validationErr != nil {
		return validationErr
	}

	params := [][]byte{
		[]byte(opFn),
		[]byte(right.Channel),
		[]byte(right.Chaincode),
		[]byte(right.Role),
		[]byte(right.Operation),
		[]byte(right.Address),
	}
	aclstub := w.ledger.GetStub(chNameACL)
	aclstub.TxID = txIDGen()
	aclstub.MockPeerChaincodeWithChannel(chNameACL, aclstub, chNameACL)

	rsp := aclstub.InvokeChaincode(chNameACL, params, chNameACL)
	if rsp.GetStatus() != shim.OK {
		return errors.New(rsp.GetMessage())
	}

	return nil
}

// Deprecated: use package ../mocks instead
// AddAddressRightForNominee adds right to transfer from specified principal address for nominee
func (w *Wallet) AddAddressRightForNominee(right *AddressRight) error {
	return w.modifyAddressRightForNominee(FnAddAddressRightForNominee, right)
}

// Deprecated: use package ../mocks instead
// RemoveAddressRightFromNominee removes right to transfer from specified principal address from nominee
func (w *Wallet) RemoveAddressRightFromNominee(right *AddressRight) error {
	return w.modifyAddressRightForNominee(FnRemoveAddressRightFromNominee, right)
}

// Deprecated: use package ../mocks instead
func (w *Wallet) modifyAddressRightForNominee(opFn operation, right *AddressRight) error {
	if right == nil {
		return errors.New(ErrRightNotSet)
	}

	validationErr := right.IsValid()
	if validationErr != nil {
		return validationErr
	}

	params := [][]byte{
		[]byte(opFn),
		[]byte(right.Channel),
		[]byte(right.Chaincode),
		[]byte(right.NomineeAddress),
		[]byte(right.PrincipalAddress),
	}
	aclstub := w.ledger.GetStub(chNameACL)
	aclstub.TxID = txIDGen()
	aclstub.MockPeerChaincodeWithChannel(chNameACL, aclstub, chNameACL)

	rsp := aclstub.InvokeChaincode(chNameACL, params, chNameACL)
	if rsp.GetStatus() != shim.OK {
		return errors.New(rsp.GetMessage())
	}

	return nil
}
