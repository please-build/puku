package generate

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/please"
	"github.com/please-build/puku/proxy"
	"github.com/please-build/puku/trie"
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

	trgt, err := u.localDep("test_project/foo")
	require.NoError(t, err)
	assert.Equal(t, "//test_project/foo:bar", trgt)

	trgt, err = u.localDep("github.com/some/module/test_project/foo")
	require.NoError(t, err)
	assert.Equal(t, "//test_project/foo:bar", trgt)
}

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

func TestResolveImport(t *testing.T) {
	installs := trie.New()
	installs.Add("installed", "//third_party/go:installed")

	conf := &config.Config{
		KnownTargets: map[string]string{
			"knowntarget": "//third_party/go:known_target",
		},
	}

	u := Update{
		plzConf:  &please.Config{},
		installs: installs,
		resolvedImports: map[string]string{
			"resolved": "//third_party/go:resolved",
		},
		modules: []string{"github.com/cached-module"},
		proxy:   proxy.New(proxy.DefaultURL),
	}

	t.Run("resolve against a module in the puku.json", func(t *testing.T) {
		ret, err := u.resolveImport(conf, "knowntarget")
		require.NoError(t, err)
		assert.Equal(t, "//third_party/go:known_target", ret)
	})

	t.Run("resolve against a module that we already know about", func(t *testing.T) {
		ret, err := u.resolveImport(conf, "resolved")
		require.NoError(t, err)
		assert.Equal(t, "//third_party/go:resolved", ret)
	})

	t.Run("resolve against module install list", func(t *testing.T) {
		ret, err := u.resolveImport(conf, "installed")
		require.NoError(t, err)
		assert.Equal(t, "//third_party/go:installed", ret)
	})

	t.Run("resolve against cached module name", func(t *testing.T) {
		ret, err := u.resolveImport(conf, "github.com/cached-module/package")
		require.NoError(t, err)
		assert.Equal(t, "///third_party/go/github.com_cached-module//package", ret)
	})

	t.Run("resolve against the module proxy", func(t *testing.T) {
		ret, err := u.resolveImport(conf, "github.com/please-build/puku/package")
		require.NoError(t, err)
		assert.Equal(t, "///third_party/go/github.com_please-build_puku//package", ret)
	})
}
