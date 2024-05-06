package version_test

import (
	"testing"

	"github.com/anoideaopen/foundation/version"
	"github.com/stretchr/testify/require"
)

func TestBuildInfo(t *testing.T) {
	bi, err := version.BuildInfo()
	require.NoError(t, err)
	require.NotNil(t, bi)
}
