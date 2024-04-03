package core

// ContractOptions
// TxTTL - Transaction time to live in seconds. By default, 0 means an eternal life.
// Checked during batch execution. In the US, it is set to 30 seconds.
// BatchPrefix - the prefix with which preimages are stored in HLF. By default, it's "batchTransactions."
// The US sets its shorter prefix from one or two characters.
// NonceTTL - time in seconds for nonce. If we attempt to execute a transaction in a batch
// that is older than the maximum nonce (at the current moment) by more than NonceTTL,
// we will not execute it and receive an error. In the US, it is set to 50 seconds.
// If NonceTTL = 0, then the check is done "the old way" when adding preimages.
// IsOtherNoncePrefix - historically, US uses a different prefix for nonces.
// We are obligated to support different prefixes, but it's not worth creating more of them. Therefore, it's only a flag.

// ContractOptions is a struct for contract options
type ContractOptions struct {
	DisabledFunctions  []string
	DisableSwaps       bool
	DisableMultiSwaps  bool
	TxTTL              uint
	BatchPrefix        string
	NonceTTL           uint
	IsOtherNoncePrefix bool
}
