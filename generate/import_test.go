package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportDir(t *testing.T) {
	fooDir, err := ImportDir("test_project/foo")
	require.NoError(t, err)

	foo := fooDir["foo.go"]
	fooTest := fooDir["foo_test.go"]
	externalTest := fooDir["external_test.go"]

	require.NotNil(t, foo)
	require.NotNil(t, fooTest)
	require.NotNil(t, externalTest)

	assert.Equal(t, foo.Imports(), []string{"github.com/example/module"})
	assert.Equal(t, fooTest.Imports(), []string{"github.com/stretchr/testify/assert"})
	assert.Equal(t, externalTest.Imports(), []string{"github.com/stretchr/testify/require"})

	assert.False(t, foo.IsTest())
	assert.True(t, fooTest.IsTest())
	assert.True(t, externalTest.IsTest())

	assert.False(t, foo.IsExternal("foo"))
	assert.False(t, fooTest.IsExternal("foo"))
	assert.True(t, externalTest.IsExternal("foo"))

	assert.False(t, foo.IsCmd())
	assert.False(t, fooTest.IsCmd())
	assert.False(t, externalTest.IsCmd())

	mainDir, err := ImportDir("test_project")
	require.NoError(t, err)

	main := mainDir["main.go"]
	require.NotNil(t, main)

	require.True(t, main.IsCmd())
	require.False(t, main.IsTest())
	require.False(t, main.IsExternal("test_project"))
}

func TestImportTSDir(t *testing.T) {
	tsDir, err := ImportDir("test_project/ts")
	require.NoError(t, err)

	foo := tsDir["foo.ts"]
	bar := tsDir["bar.ts"]

	require.NotNil(t, foo)
	require.NotNil(t, bar)

	assert.Equal(t, foo.Imports(), []string{"react", "./bar", "./lazy_loaded"})
	assert.Equal(t, bar.Imports(), []string{})

	assert.False(t, foo.IsTest())
	assert.False(t, bar.IsTest())
}
