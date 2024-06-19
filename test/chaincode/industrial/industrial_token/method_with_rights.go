package industrialtoken

import (
	"errors"

	"github.com/anoideaopen/foundation/core/acl"
	"github.com/anoideaopen/foundation/core/types"
)

var ErrUnauthorized = errors.New("unauthorized")

func (it *IndustrialToken) TxMethodWithRights(sender *types.Sender) error {
	if err := it.checkIfIssuer(sender.Address()); err != nil {
		return err
	}

	return ErrUnauthorized
}

func (it *IndustrialToken) checkIfIssuer(address *types.Address) error {
	params := []string{it.GetStub().GetChannelID(), it.GetID(), acl.Issuer.String(), "", address.String()}
	haveRight, err := acl.GetAccountRight(it.GetStub(), params)
	if err != nil {
		return err
	}

	if haveRight != nil && !haveRight.GetHaveRight() {
		return ErrUnauthorized
	}

	return nil
}
