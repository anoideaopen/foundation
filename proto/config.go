package proto

// IsAdminSet checks whether the contract admin wallet is set in the ContractConfig.
func (bc *ContractConfig) IsAdminSet() bool {
	return bc.Admin != nil && bc.Admin.Address != ""
}
