package proto

import "github.com/hyperledger/fabric-chaincode-go/v2/shim"

//go:generate protoc -I=. --go_out=paths=source_relative:. task.proto
//go:generate protoc -I=. --go_out=paths=source_relative:. batch.proto
//go:generate protoc -I=. --go_out=paths=source_relative:. report.proto
//go:generate protoc -I=. --go_out=paths=source_relative:. locks.proto
//go:generate protoc -I=. -I=./validate --go_out=paths=source_relative:. --validate_out=lang=go,paths=source_relative:. transfer_request.proto

// Chaincode configuration
//go:generate protoc -I=. -I=./validate --go_out=paths=source_relative:. --validate_out=lang=go,paths=source_relative:. foundation_config.proto

//go:generate counterfeiter -generate

//counterfeiter:generate -o ../mocks/chaincode_stub.go --fake-name ChaincodeStub . chaincodeStub
type chaincodeStub interface { //nolint:unused
	shim.ChaincodeStubInterface
}

//counterfeiter:generate -o ../mocks/state_iterator.go --fake-name StateIterator . stateIterator
type stateIterator interface { //nolint:unused
	shim.StateQueryIteratorInterface
}
