package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/core/acl"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

type IssuerCheckerToken struct {
	token.BaseToken
}

const (
	fnGetRight                   = "getRight"
	fnGetAddressRightForNominee  = "getAddressRightForNominee"
	fnGetAddressesListForNominee = "getAddressesListForNominee"
)

func (ict *IssuerCheckerToken) QueryGetRight(ccname string, address *types.Address, role, operation string) (bool, error) {
	stub := ict.GetStub()
	r := acl.Role(role)
	params := []string{stub.GetChannelID(), ccname, r.String(), operation, address.String()}
	right, err := acl.GetAccountRight(ict.GetStub(), params)
	if err != nil {
		return false, err
	}

	if right.HaveRight {
		return true, nil
	}

	return false, err
}

func (ict *IssuerCheckerToken) QueryGetAddressRightForNominee(chaincodeName string, nomineeAddress, principalAddress *types.Address) (bool, error) {
	stub := ict.GetStub()
	params := []string{stub.GetChannelID(), chaincodeName, nomineeAddress.String(), principalAddress.String()}
	right, err := acl.GetAddressRightForNominee(ict.GetStub(), params)
	if err != nil {
		return false, err
	}

	if right.HaveRight {
		return true, nil
	}

	return false, err
}

func (ict *IssuerCheckerToken) QueryGetAddressesListForNominee(chaincodeName string, nomineeAddress *types.Address) ([]*types.Address, error) {
	stub := ict.GetStub()
	params := []string{stub.GetChannelID(), chaincodeName, nomineeAddress.String()}
	accounts, err := acl.GetAddressesListForNominee(ict.GetStub(), params)
	if err != nil {
		return nil, err
	}

	var addresses []*types.Address
	for _, addr := range accounts.GetAddresses() {
		addresses = append(addresses, &types.Address{
			UserID:       addr.UserID,
			Address:      addr.Address,
			IsIndustrial: addr.IsIndustrial,
			IsMultisig:   addr.IsMultisig,
		})
	}

	return addresses, nil
}

func TestRights(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	issuer := ledgerMock.NewWallet()

	config := makeBaseTokenConfig("NT Token", "NT", 8,
		issuer.Address(), "", "", "", nil)

	initMsg := ledgerMock.NewCC(testTokenCCName, &IssuerCheckerToken{}, config)
	require.Empty(t, initMsg)

	const (
		createOp = "createEmissionApp"
		acceptOp = "acceptEmissionApp"
		deleteOp = "deleteEmissionApp"
	)

	user := ledgerMock.NewWallet()

	t.Run("add right & check if it's granted for user and operation", func(t *testing.T) {
		err := issuer.AddAccountRight(&mock.Right{
			Channel:   testTokenCCName,
			Chaincode: testTokenCCName,
			Role:      acl.Issuer.String(),
			Operation: createOp,
			Address:   user.Address(),
		})
		require.NoError(t, err)

		isIssuer := issuer.Invoke(testTokenCCName, fnGetRight,
			testTokenCCName, user.Address(), acl.Issuer.String(), createOp)
		require.Equal(t, "true", isIssuer)
	})

	t.Run("multi-emission, non-permitted operation", func(t *testing.T) {
		isIssuer := issuer.Invoke(testTokenCCName, fnGetRight,
			testTokenCCName, issuer.Address(), acl.Issuer.String(), deleteOp)
		require.Equal(t, "false", isIssuer)
	})

	t.Run("remove right & check it is removed", func(t *testing.T) {
		err := issuer.RemoveAccountRight(&mock.Right{
			Channel:   testTokenCCName,
			Chaincode: testTokenCCName,
			Role:      acl.Issuer.String(),
			Operation: createOp,
			Address:   user.Address(),
		})
		require.NoError(t, err)
		isIssuer := issuer.Invoke(testTokenCCName, fnGetRight,
			testTokenCCName, user.Address(), acl.Issuer.String(), createOp)
		require.Equal(t, "false", isIssuer)
	})

	t.Run("check double setting right", func(t *testing.T) {
		err := issuer.AddAccountRight(&mock.Right{
			Channel:   testTokenCCName,
			Chaincode: testTokenCCName,
			Role:      acl.Issuer.String(),
			Operation: acceptOp,
			Address:   user.Address(),
		})
		require.NoError(t, err)

		err = issuer.AddAccountRight(&mock.Right{
			Channel:   testTokenCCName,
			Chaincode: testTokenCCName,
			Role:      acl.Issuer.String(),
			Operation: acceptOp,
			Address:   user.Address(),
		})
		require.NoError(t, err)

		err = issuer.RemoveAccountRight(&mock.Right{
			Channel:   testTokenCCName,
			Chaincode: testTokenCCName,
			Role:      acl.Issuer.String(),
			Operation: acceptOp,
			Address:   user.Address(),
		})
		require.NoError(t, err)

		isIssuer := issuer.Invoke(testTokenCCName, fnGetRight,
			testTokenCCName, user.Address(), acl.Issuer.String(), acceptOp)
		require.Equal(t, "false", isIssuer)
	})
}

func TestAddressRightsForNominee(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	issuer := ledgerMock.NewWallet()

	config := makeBaseTokenConfig("NT Token", "NT", 8,
		issuer.Address(), "", "", "", nil)

	initMsg := ledgerMock.NewCC(testTokenCCName, &IssuerCheckerToken{}, config)
	require.Empty(t, initMsg)

	nominee := ledgerMock.NewWallet()
	principal := ledgerMock.NewWallet()
	user := ledgerMock.NewWallet()

	t.Run("add right & check if it's granted for nominee", func(t *testing.T) {
		err := issuer.AddAddressRightForNominee(&mock.AddressRight{
			Channel:          testTokenCCName,
			Chaincode:        testTokenCCName,
			NomineeAddress:   nominee.Address(),
			PrincipalAddress: principal.Address(),
		})
		require.NoError(t, err)

		haveRight := issuer.Invoke(testTokenCCName, fnGetAddressRightForNominee,
			testTokenCCName, nominee.Address(), principal.Address())
		require.Equal(t, "true", haveRight)
	})

	t.Run("[negative] requesting right for another user", func(t *testing.T) {
		haveRight := issuer.Invoke(testTokenCCName, fnGetAddressRightForNominee,
			testTokenCCName, nominee.Address(), user.Address())
		require.Equal(t, "false", haveRight)
	})

	t.Run("remove right & check it is removed", func(t *testing.T) {
		err := issuer.RemoveAddressRightFromNominee(&mock.AddressRight{
			Channel:          testTokenCCName,
			Chaincode:        testTokenCCName,
			NomineeAddress:   nominee.Address(),
			PrincipalAddress: principal.Address(),
		})
		require.NoError(t, err)
		haveRight := issuer.Invoke(testTokenCCName, fnGetAddressRightForNominee,
			testTokenCCName, nominee.Address(), principal.Address())
		require.Equal(t, "false", haveRight)
	})

	t.Run("check double setting right", func(t *testing.T) {
		err := issuer.AddAddressRightForNominee(&mock.AddressRight{
			Channel:          testTokenCCName,
			Chaincode:        testTokenCCName,
			NomineeAddress:   nominee.Address(),
			PrincipalAddress: principal.Address(),
		})
		require.NoError(t, err)

		err = issuer.AddAddressRightForNominee(&mock.AddressRight{
			Channel:          testTokenCCName,
			Chaincode:        testTokenCCName,
			NomineeAddress:   nominee.Address(),
			PrincipalAddress: principal.Address(),
		})
		require.NoError(t, err)

		rawAddresses := issuer.Invoke(testTokenCCName, fnGetAddressesListForNominee,
			testTokenCCName, nominee.Address())
		require.NoError(t, err)
		require.Equal(t, "[\""+principal.Address()+"\"]", rawAddresses)
	})
}
