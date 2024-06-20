package industrialtoken

import (
	"errors"
	"strings"

	"github.com/anoideaopen/foundation/core/acl"
	"github.com/anoideaopen/foundation/core/types"
)

var ErrUnauthorized = errors.New("unauthorized")

func (it *IndustrialToken) TxWithRights(sender *types.Sender) error {
	if err := it.checkIfIssuer(sender.Address()); err != nil {
		return err
	}

	return nil
}

func (it *IndustrialToken) checkIfIssuer(address *types.Address) error {
	params := []string{strings.ToLower(it.GetStub().GetChannelID()), strings.ToLower(it.GetID()), acl.Issuer.String(), "", address.String()}
	haveRight, err := acl.GetAccountRight(it.GetStub(), params)
	if err != nil {
		return err
	}

	if haveRight != nil && !haveRight.GetHaveRight() {
		return ErrUnauthorized
	}

	return nil
}
