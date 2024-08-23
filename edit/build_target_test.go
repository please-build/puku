package edit

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildTarget(t *testing.T) {
	local := BuildTarget("foo", "", "")
	assert.Equal(t, local, ":foo")

	root := BuildTarget("foo", ".", "")
	assert.Equal(t, "//:foo", root)

	pkg := BuildTarget("foo", "pkg", "")
	assert.Equal(t, "//pkg:foo", pkg)

	pkgSameName := BuildTarget("foo", "foo", "")
	assert.Equal(t, "//foo", pkgSameName)

	subrepo := BuildTarget("foo", "pkg", "repo")
	assert.Equal(t, "///repo//pkg:foo", subrepo)

	subrepoRoot := BuildTarget("foo", ".", "repo")
	assert.Equal(t, "///repo//:foo", subrepoRoot)

	subrepoRootAlt := BuildTarget("foo", "", "repo")
	assert.Equal(t, "///repo//:foo", subrepoRootAlt)
}
