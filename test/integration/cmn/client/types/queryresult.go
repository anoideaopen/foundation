package types

import (
	"encoding/json"
	"fmt"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

type QueryResult struct {
	txID      string
	errorCode int32
	response  []byte
	message   []byte
}

// Getters

func (qr *QueryResult) TxID() string {
	return qr.txID
}

func (qr *QueryResult) RawResult() ([]byte, []byte) {
	return qr.response, qr.message
}

func (qr *QueryResult) ErrorCode() int32 {
	return qr.errorCode
}

// Setters

func (qr *QueryResult) SetTxID(txID string) {
	qr.txID = txID
}

func (qr *QueryResult) SetMessage(message []byte) {
	qr.message = message
}

func (qr *QueryResult) SetResponse(response []byte) {
	qr.response = response
}

func (qr *QueryResult) SetErrorCode(errorCode int32) {
	qr.errorCode = errorCode
}

// Checkers

func (qr *QueryResult) CheckResultEquals(reference string) {
	checkResult := func() string {
		gomega.Expect(qr.checkErrIsNil()).Should(gomega.BeEmpty())

		if string(qr.response) != reference {
			return "response message not equals to expected"
		}

		return ""
	}

	gomega.Expect(checkResult()).Should(gomega.BeEmpty())
}

func (qr *QueryResult) CheckResultContains(reference string) {
	gomega.Expect(qr.checkErrIsNil()).Should(gomega.BeEmpty())
	gomega.Expect(gbytes.BufferWithBytes(qr.response)).Should(gbytes.Say(reference))
}

func (qr *QueryResult) CheckErrorEquals(errMessage string) {
	checkResult := func() string {
		if errMessage == "" {
			return qr.checkErrIsNil()
		}

		gomega.Expect(gbytes.BufferWithBytes(qr.message)).To(gbytes.Say(errMessage))
		return ""
	}

	gomega.Expect(checkResult()).Should(gomega.BeEmpty())
}

func (qr *QueryResult) CheckErrorIsNil() {
	gomega.Expect(qr.checkErrIsNil()).Should(gomega.BeEmpty())
}

// non-interface based functions

func (qr *QueryResult) CheckBalance(expectedBalance string) {
	checkResult := func() string {
		gomega.Expect(qr.checkErrIsNil()).Should(gomega.BeEmpty())

		response := string(qr.response[:len(qr.response)-1]) // skip line feed

		if response != "\""+expectedBalance+"\"" {
			return fmt.Sprintf("actual balance: %s not equals to expected: %s", response, expectedBalance)
		}

		return ""
	}

	gomega.Expect(checkResult()).Should(gomega.BeEmpty())
}

func (qr *QueryResult) CheckIndustrialBalance(expectedGroup, expectedBalance string) {
	checkResult := func() string {
		m := make(map[string]string)
		err := json.Unmarshal(qr.response, &m)
		if err != nil {
			return fmt.Sprintf("error unmarshalling json: %v, source '%s", err, string(qr.response))
		}
		v, ok := m[expectedGroup]
		if !ok {
			v = "0"
		}
		if v != expectedBalance {
			return fmt.Sprintf("group balance of '%s' with balance '%s' not eq '%s' expected amount", expectedGroup, v, expectedBalance)
		}
		return ""
	}

	gomega.Expect(checkResult()).Should(gomega.BeEmpty())
}

func (qr *QueryResult) CheckResponseWithFunc(responseCheckFunc func([]byte) string) {
	gomega.Expect(qr.checkErrIsNil()).Should(gomega.BeEmpty())
	gomega.Expect(responseCheckFunc(qr.response)).Should(gomega.BeEmpty())
}

func (qr *QueryResult) checkErrIsNil() string {
	if qr.errorCode == 0 && qr.message == nil {
		return ""
	}

	if qr.errorCode != 0 && qr.message != nil {
		return "error message: " + string(qr.message)
	}

	return ""
}
