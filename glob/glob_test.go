package glob

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGlob(t *testing.T) {
	t.Run("globs go files only", func(t *testing.T) {
		files, err := Glob("glob/test_data", []string{"*_test.go"}, nil)
		require.NoError(t, err)

		assert.Equal(t, []string{"bar_test.go"}, files)
	})

	t.Run("excludes pattern", func(t *testing.T) {
		files, err := Glob("glob/test_data", []string{"*.go"}, []string{"*_test.go"})
		require.NoError(t, err)

		assert.Equal(t, []string{"bar.go"}, files)
	})
}