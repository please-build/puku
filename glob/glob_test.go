package glob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlob(t *testing.T) {
	g := &Globber{cache: map[pattern][]string{}}
	t.Run("globs go files only", func(t *testing.T) {
		files, err := g.Glob("test_project", &Args{
			Include: []string{"*_test.go"},
		})
		require.NoError(t, err)

		assert.ElementsMatch(t, []string{"bar_test.go"}, files)
	})

	t.Run("excludes pattern", func(t *testing.T) {
		files, err := g.Glob("test_project", &Args{
			Include: []string{"*.go"},
			Exclude: []string{"*_test.go"},
		})
		require.NoError(t, err)

		assert.ElementsMatch(t, []string{"main.go", "bar.go"}, files)
	})
}
