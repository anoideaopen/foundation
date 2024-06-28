package version_test

import (
	"testing"

	"github.com/anoideaopen/foundation/version"
	"github.com/stretchr/testify/require"
)

func TestSystemEnv(t *testing.T) {
	s := version.SystemEnv()
	_, ok := s["/etc/issue"]
	require.True(t, ok)
}
