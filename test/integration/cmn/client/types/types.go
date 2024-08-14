package types

import (
	"fmt"
)

type Reference string

type Result interface {
	TxId() string
	RawResult()
	CheckResultEquals(reference *Reference)
	CheckErrorEquals(reference *Reference)
}

func compareResultToReference(actual []byte, reference string) error {
	result := string(actual)
	if reference == "" && actual != nil {
		return fmt.Errorf("result was expected to be nil but equals to %s", result)
	}

	if reference != "" && actual == nil {
		return fmt.Errorf("result was expected to be %v but is nil", reference)
	}

	if reference != result {
		return fmt.Errorf("result was expected to be %v but equals to %s", reference, result)
	}

	return nil
}
