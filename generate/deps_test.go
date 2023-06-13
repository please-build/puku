package generate

import (
	"github.com/please-build/puku/please"
	"path/filepath"
	"testing"

	"github.com/please-build/puku/config"
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

	trgt, err := u.localDep(new(config.Config), "generate/test_data/foo")
	require.NoError(t, err)
	assert.Equal(t, "//generate/test_data/foo:bar", trgt)

	trgt, err = u.localDep(new(config.Config), "github.com/some/module/generate/test_data/foo")
	require.NoError(t, err)
	assert.Equal(t, "//generate/test_data/foo:bar", trgt)
}
