package reflectx

// BytesEncoder defines an interface for encoding an object to bytes.
type BytesEncoder interface {
	EncodeToBytes() ([]byte, error)
}

// BytesDecoder defines an interface for decoding an object from bytes.
type BytesDecoder interface {
	DecodeFromBytes([]byte) error
}
