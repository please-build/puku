package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var buildFileNames = []string{"BUILD_FILE", "BUILD_FILE.plz"}

func TestParseBuildFile(t *testing.T) {
	f, err := parseBuildFile("generate/test_data", buildFileNames)
	require.NoError(t, err)

	libs := f.Rules("go_library")
	require.Len(t, libs, 1)

	f, err = parseBuildFile("generate/test_data/foo", buildFileNames)
	require.NoError(t, err)

	libs = f.Rules("go_library")
	require.Len(t, libs, 1)

	f, err = parseBuildFile("generate/test_data/foo/bar", buildFileNames)
	require.NoError(t, err)
	assert.Equal(t, "generate/test_data/foo/bar/BUILD_FILE", f.Path)
}
