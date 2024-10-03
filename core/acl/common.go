package acl

// ChACL - ACL channel name,
// CcACL - ACL chaincode name
const (
	ChACL = "acl"
	CcACL = "acl"
)

// acl chaincode functions
const (
	FnCheckAddress                  = "checkAddress"
	FnCheckKeys                     = "checkKeys"
	FnGetAccountInfo                = "getAccountInfo"
	FnGetAccountsInfo               = "getAccountsInfo"
	FnGetAccOpRight                 = "getAccountOperationRight"
	FnAddRights                     = "addRights"
	FnRemoveRights                  = "removeRights"
	FnAddAddressRightForNominee     = "addAddressRightForNominee"
	FnRemoveAddressRightFromNominee = "removeAddressRightFromNominee"
	FnGetAddressRightForNominee     = "getAddressRightForNominee"
	FnGetAddressesListForNominee    = "getAddressesListForNominee"
)
