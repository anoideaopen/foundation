package types

type Reference string

type Setters interface {
	SetTxID(txID string)
	SetResult(response, message string, errorCode int32)
}

type Getters interface {
	TxID() string
	RawResult() ([]byte, []byte)
	ErrorCode() int32
}

type ResultInterface interface {
	Setters
	Getters
	CheckResultEquals(reference string)
	CheckErrorEquals(errMessage string)
	CheckErrorIsNil()
}
