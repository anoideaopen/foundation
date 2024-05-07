package version

import (
	"errors"
	"runtime/debug"
)

// BuildInfo returns the build information
func BuildInfo() (*debug.BuildInfo, error) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, errors.New("fetching build info failed")
	}

	if bi == nil {
		return nil, errors.New("build information is empty")
	}

	return bi, nil
}
