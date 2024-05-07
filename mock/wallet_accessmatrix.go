package mock

import (
	"errors"

	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// Right represents a right in the access matrix
type Right struct {
	Channel   string
	Chaincode string
	Role      string
	Operation string
	Address   string
}

// IsValid checks if the right is valid
func (r Right) IsValid() error {
	if len(r.Channel) == 0 {
		return errors.New("right is broken, channel is not set")
	}

	if len(r.Chaincode) == 0 {
		return errors.New("right is broken, chaincode is not set")
	}

	if len(r.Role) == 0 {
		return errors.New("right is broken, role is not set")
	}

	if len(r.Address) == 0 {
		return errors.New("right is broken, address is not set")
	}

	return nil
}

type operation string

const (
	// AddRights adds a right to the access matrix
	AddRights operation = "addRights"
	// RemoveRights removes a right from the access matrix
	RemoveRights operation = "removeRights"
)

// AddAccountRight adds a right to the access matrix
func (w *Wallet) AddAccountRight(right *Right) error {
	return w.modifyRight(AddRights, right)
}

// RemoveAccountRight removes a right from the access matrix
func (w *Wallet) RemoveAccountRight(right *Right) error {
	return w.modifyRight(RemoveRights, right)
}

func (w *Wallet) modifyRight(opFn operation, right *Right) error {
	if right == nil {
		return errors.New("right is not set")
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
	const acl = "acl"
	aclstub := w.ledger.GetStub(acl)
	aclstub.TxID = txIDGen()
	aclstub.MockPeerChaincodeWithChannel(acl, aclstub, acl)

	rsp := aclstub.InvokeChaincode(acl, params, acl)
	if rsp.GetStatus() != shim.OK {
		return errors.New(rsp.GetMessage())
	}

	return nil
}
