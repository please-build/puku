package work

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandPaths(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	ret, err := ExpandPaths(".", []string{"foo"})
	require.NoError(t, err)
	assert.ElementsMatch(t, ret, []string{"foo"})

	ret, err = ExpandPaths("foo", []string{"bar"})
	require.NoError(t, err)
	assert.ElementsMatch(t, ret, []string{"foo/bar"})

	ret, err = ExpandPaths("foo", []string{"bar"})
	require.NoError(t, err)
	assert.ElementsMatch(t, ret, []string{"foo/bar"})

	ret, err = ExpandPaths(".", []string{filepath.Join(wd, "bar")})
	require.NoError(t, err)
	assert.ElementsMatch(t, ret, []string{"bar"})

	ret, err = ExpandPaths("foo", []string{filepath.Join(wd, "bar")})
	require.NoError(t, err)
	assert.ElementsMatch(t, ret, []string{"bar"})
}
