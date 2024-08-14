package types

type QueryResult struct {
	txID     string
	response []byte
	message  []byte
}

func (qr *QueryResult) TxID() string {
	return qr.txID
}

func (qr *QueryResult) RawResult() ([]byte, []byte) {
	return qr.response, qr.message
}

func (qr *QueryResult) CheckResultEquals(reference *Reference) bool {
	return false
}

func (qr *QueryResult) CheckBalance(expectedBalance *Reference) bool {
	return false
}

func (qr *QueryResult) CheckIndustrialBalance(expectedGroup, expectedBalance *Reference) bool {
	return false
}

func (qr *QueryResult) CheckErrorEquals(reference *Reference) {
	/*
		var result Reference
		result =
		err := compareResultToReference(string(qr.message), reference)
		Expect(err).NotTo(HaveOccured())
	*/

	return false
}
