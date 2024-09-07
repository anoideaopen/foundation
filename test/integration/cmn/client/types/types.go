package types

type Setters interface {
	SetTxID(string)
	SetResponse(response []byte)
	SetMessage(message []byte)
	SetErrorCode(errorCode int32)
}

type Getters interface {
	TxID() string
	RawResult() ([]byte, []byte)
	ErrorCode() int32
}

type Checkers interface {
	CheckResultEquals(reference string)
	CheckResultContains(reference string)
	CheckErrorEquals(errMessage string)
	CheckErrorIsNil()
}

type ResultInterface interface {
	Setters
	Getters
	Checkers
}
