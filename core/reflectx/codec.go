package reflectx

import "github.com/hyperledger/fabric-chaincode-go/shim"

// BytesEncoder defines an interface for encoding an object to bytes.
type BytesEncoder interface {
	EncodeToBytes() ([]byte, error)
}

// BytesDecoder defines an interface for decoding an object from bytes.
type BytesDecoder interface {
	DecodeFromBytes([]byte) error
}

// BytesEncoder defines an interface for encoding an object to bytes with a stub.
type StubBytesEncoder interface {
	EncodeToBytesWithStub(shim.ChaincodeStubInterface) ([]byte, error)
}

// BytesDecoder defines an interface for decoding an object from bytes with a stub.
type StubBytesDecoder interface {
	DecodeFromBytesWithStub(shim.ChaincodeStubInterface, []byte) error
}
