package client

import (
	"encoding/json"
	"fmt"
)

func CheckResult(successF func(out []byte) string, errorF func(outErr []byte) string) CheckResultFunc {
	return func(err error, sessExitCode int, sessError []byte, sessOut []byte) string {
		if (successF == nil && errorF == nil) ||
			(successF != nil && errorF != nil) {
			return "error: only one function must be defined"
		}

		if err != nil {
			return fmt.Sprintf("error executing command: %v", err)
		}

		if successF != nil {
			if sessExitCode != 0 {
				return fmt.Sprintf("exit code is %d: %s, %v", sessExitCode, string(sessError), err)
			}
			out := sessOut[:len(sessOut)-1] // skip line feed
			return successF(out)
		}

		if sessExitCode == 0 {
			return fmt.Sprintf("exit code is %d", sessExitCode)
		}

		return errorF(sessError)
	}
}

func CheckBalance(etalon string) func([]byte) string {
	return func(out []byte) string {
		etl := "\"" + etalon + "\""
		if string(out) != etl {
			return "not equal " + string(out) + " and " + etl
		}
		return ""
	}
}

func CheckIndustrialBalance(expectedGroup string, expectedAmount string) func([]byte) string {
	return func(out []byte) string {
		m := make(map[string]string)
		err := json.Unmarshal(out, &m)
		if err != nil {
			return fmt.Sprintf("error unmarshalling json: %v, source '%s", err, string(out))
		}
		v, ok := m[expectedGroup]
		if !ok {
			v = "0"
		}
		if v != expectedAmount {
			return fmt.Sprintf("group balance of '%s' with balance '%s' not eq '%s' expected amount", expectedGroup, v, expectedAmount)
		}
		return ""
	}
}

func CheckTxResponseResult(expectedErrorMsg string) func([]byte) string {
	return func(out []byte) string {
		occurredError := string(out)

		if occurredError != expectedErrorMsg {
			return fmt.Sprintf("expected error '%s' not equals to occurred error '%s'", expectedErrorMsg, occurredError)
		}
		return ""
	}
}
