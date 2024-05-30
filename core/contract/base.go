package contract

// Base is the minimal interface required for a contract to execute within the system.
type Base interface {
	ID() string // ID retrieves the unique identifier for the contract.

	Configurator
	StubGetSetter
}
