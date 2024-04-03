package version_test

import (
	"testing"

	"github.com/anoideaopen/foundation/version"
	"github.com/stretchr/testify/assert"
)

func TestSystemEnv(t *testing.T) {
	s := version.SystemEnv()
	_, ok := s["/etc/issue"]
	assert.True(t, ok)
}
