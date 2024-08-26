package types

import (
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"strings"
)

type InvokeResult struct {
	txID      string
	errorCode int32
	response  []byte
	message   []byte
}

func (ir *InvokeResult) TxID() string {
	gomega.Expect(ir.checkErrIsNil()).Should(gomega.BeEmpty())
	return ir.txID
}

func (ir *InvokeResult) SetTxID(txID string) {
	ir.txID = txID
}

func (ir *InvokeResult) SetErrorCode(errorCode int32) {
	ir.errorCode = errorCode
}

func (ir *InvokeResult) ErrorCode() int32 {
	return ir.errorCode
}

func (ir *InvokeResult) SetMessage(message []byte) {
	ir.message = message
}

func (ir *InvokeResult) SetResponse(response []byte) {
	ir.response = response
}

func (ir *InvokeResult) RawResult() ([]byte, []byte) {
	return ir.response, ir.message
}

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

func (ir *InvokeResult) CheckErrorIsNil() {
	gomega.Expect(ir.checkErrIsNil()).Should(gomega.BeEmpty())
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

func (ir *InvokeResult) checkErrIsNil() string {
	if ir.errorCode == 0 && ir.message == nil {
		return ""
	}

	errMsg := strings.Split(string(ir.message), "Error")[1]

	if ir.errorCode != 0 && ir.message != nil {
		return "error occurred: " + errMsg
	}

	return ""
}
