package generate

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestDepTarget(t *testing.T) {
	exampleModule := "github.com/example/module"
	modules := []string{exampleModule, filepath.Join(exampleModule, "foo")}

	t.Run("returns longest match", func(t *testing.T) {
		label := depTarget(modules, filepath.Join(exampleModule, "foo", "bar"), "third_party/go")
		assert.Equal(t, "///third_party/go/github.com_example_module_foo//bar:bar", label)
	})

	t.Run("returns root package", func(t *testing.T) {
		label := depTarget(modules, exampleModule, "third_party/go")
		assert.Equal(t, "///third_party/go/github.com_example_module//:module", label)
	})
}


func TestLocalDeps(t *testing.T) {
	u := &Update{
		buildFileNames: buildFileNames,
		kinds: defaultKinds,
	}
	trgt, err := u.localDep("generate/test_data/foo")
	require.NoError(t, err)
	assert.Equal(t, "//generate/test_data/foo:bar", trgt)

	u.importPath = "github.com/some/module"
	trgt, err = u.localDep("github.com/some/module/generate/test_data/foo")
	require.NoError(t, err)
	assert.Equal(t, "//generate/test_data/foo:bar", trgt)
}