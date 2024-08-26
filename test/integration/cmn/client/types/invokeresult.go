package types

import (
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

type InvokeResult struct {
	txID      string
	errorCode int32
	response  []byte
	message   []byte
}

// Getters

func (ir *InvokeResult) TxID() string {
	gomega.Expect(ir.checkErrIsNil()).Should(gomega.BeEmpty())
	return ir.txID
}

func (ir *InvokeResult) RawResult() ([]byte, []byte) {
	return ir.response, ir.message
}

func (ir *InvokeResult) ErrorCode() int32 {
	return ir.errorCode
}

// Setters

func (ir *InvokeResult) SetTxID(txID string) {
	ir.txID = txID
}

func (ir *InvokeResult) SetMessage(message []byte) {
	ir.message = message
}

func (ir *InvokeResult) SetResponse(response []byte) {
	ir.response = response
}

func (ir *InvokeResult) SetErrorCode(errorCode int32) {
	ir.errorCode = errorCode
}

// Checkers

func (ir *InvokeResult) CheckResultEquals(reference string) {
	checkResult := func() string {
		gomega.Expect(ir.checkErrIsNil()).Should(gomega.BeEmpty())

		if string(ir.response) != reference {
			return "response message not equals to expected"
		}

		return ""
	}

	gomega.Expect(checkResult()).Should(gomega.BeEmpty())
}

func (ir *InvokeResult) CheckResultContains(reference string) {
	gomega.Expect(ir.checkErrIsNil()).Should(gomega.BeEmpty())
	gomega.Expect(gbytes.BufferWithBytes(ir.response)).Should(gbytes.Say(reference))
}

func (ir *InvokeResult) CheckErrorEquals(errMessage string) {
	checkResult := func() string {
		if errMessage == "" {
			return ir.checkErrIsNil()
		}

		gomega.Expect(gbytes.BufferWithBytes(ir.message)).To(gbytes.Say(errMessage))
		return ""
	}

	gomega.Expect(checkResult()).Should(gomega.BeEmpty())
}

func (ir *InvokeResult) CheckErrorIsNil() {
	gomega.Expect(ir.checkErrIsNil()).Should(gomega.BeEmpty())
}

func (ir *InvokeResult) checkErrIsNil() string {
	if ir.errorCode == 0 && ir.message == nil {
		return ""
	}

	if ir.errorCode != 0 && ir.message != nil {
		return "error message: " + string(ir.message)
	}

	return ""
}
