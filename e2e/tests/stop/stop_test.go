package stop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/please-build/puku/e2e/harness"
	"github.com/please-build/puku/edit"
)

// A directory with stop set in its puku.json must not stop puku processing the paths that come
// after it on the command line.
func TestStopDoesntSkipLaterPaths(t *testing.T) {
	h := harness.MustNew()
	err := h.Format("stopped", "other")
	require.NoError(t, err)

	file, err := h.ParseFile("other/BUILD_FILE.plz")
	require.NoError(t, err)

	other := edit.FindTargetByName(file, "other")
	require.NotNil(t, other, "puku didn't generate a target for other/, having been given stopped/ first")
	assert.ElementsMatch(t, []string{"other.go"}, other.AttrStrings("srcs"))

	// The stopped package should still have been left alone.
	_, err = os.Stat(filepath.Join(h.RepoRoot, "stopped", "BUILD_FILE.plz"))
	assert.True(t, os.IsNotExist(err), "puku generated a build file in a package with stop set")
}
