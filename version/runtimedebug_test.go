package version_test

import (
	"testing"

	"github.com/anoideaopen/foundation/version"
	"github.com/stretchr/testify/assert"
)

func TestBuildInfo(t *testing.T) {
	bi, err := version.BuildInfo()
	assert.NoError(t, err)
	assert.NotNil(t, bi)
}
