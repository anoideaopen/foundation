package version_test

import (
	"testing"

	"github.com/anoideaopen/foundation/version"
	"github.com/stretchr/testify/assert"
)

func TestCoreChaincodeIDNameTrue(t *testing.T) {
	s := "testtest"
	t.Setenv("CORE_CHAINCODE_ID_NAME", s)
	s1 := version.CoreChaincodeIDName()
	assert.Equal(t, s, s1)
}

func TestCoreChaincodeIDNameFalse(t *testing.T) {
	s := ""
	t.Setenv("CORE_CHAINCODE_ID_NAME", s)
	s1 := version.CoreChaincodeIDName()
	assert.Equal(t, "'CORE_CHAINCODE_ID_NAME' is empty", s1)
}
