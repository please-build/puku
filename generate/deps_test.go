package generate

import (
	"path/filepath"
	"testing"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/please"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepTarget(t *testing.T) {
	exampleModule := "github.com/example/module"
	modules := []string{exampleModule, filepath.Join(exampleModule, "foo")}

	t.Run("returns longest match", func(t *testing.T) {
		label := depTarget(modules, filepath.Join(exampleModule, "foo", "bar"), "third_party/go")
		assert.Equal(t, "///third_party/go/github.com_example_module_foo//bar", label)
	})

	t.Run("returns root package", func(t *testing.T) {
		label := depTarget(modules, exampleModule, "third_party/go")
		assert.Equal(t, "///third_party/go/github.com_example_module//:module", label)
	})

	t.Run("handles when module is prefixed but not a submodule", func(t *testing.T) {
		label := depTarget(modules, exampleModule+"-foo", "third_party/go")
		assert.Equal(t, "", label)
	})
}

func TestLocalDeps(t *testing.T) {
	conf := new(please.Config)
	conf.Parse.BuildFileName = []string{"BUILD_FILE", "BUILD_FILE.plz"}
	conf.Plugin.Go.ImportPath = []string{"github.com/some/module"}

	u := NewUpdate(false, conf)

	trgt, err := u.localDep(new(config.Config), "test_project/foo")
	require.NoError(t, err)
	assert.Equal(t, "//test_project/foo:bar", trgt)

	trgt, err = u.localDep(new(config.Config), "github.com/some/module/test_project/foo")
	require.NoError(t, err)
	assert.Equal(t, "//test_project/foo:bar", trgt)
}

func TestBuildTarget(t *testing.T) {
	local := buildTarget("foo", "", "")
	assert.Equal(t, local, ":foo")

	root := buildTarget("foo", ".", "")
	assert.Equal(t, "//:foo", root)

	pkg := buildTarget("foo", "pkg", "")
	assert.Equal(t, "//pkg:foo", pkg)

	pkgSameName := buildTarget("foo", "foo", "")
	assert.Equal(t, "//foo", pkgSameName)

	subrepo := buildTarget("foo", "pkg", "repo")
	assert.Equal(t, "///repo//pkg:foo", subrepo)

	subrepoRoot := buildTarget("foo", ".", "repo")
	assert.Equal(t, "///repo//:foo", subrepoRoot)

	subrepoRootAlt := buildTarget("foo", "", "repo")
	assert.Equal(t, "///repo//:foo", subrepoRootAlt)
}
