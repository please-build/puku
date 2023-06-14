package graph

import (
	"bytes"
	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoadBuildFile(t *testing.T) {
	g := New([]string{"BUILD_FILE", "BUILD_FILE.plz"})

	f, err := g.LoadFile("test_project")
	require.NoError(t, err)

	libs := f.Rules("go_library")
	require.Len(t, libs, 1)

	f, err = g.LoadFile("test_project/foo")
	require.NoError(t, err)

	libs = f.Rules("go_library")
	require.Len(t, libs, 1)

	f, err = g.LoadFile("test_project/foo/bar")
	require.NoError(t, err)
	assert.Equal(t, "test_project/foo/bar/BUILD_FILE", f.Path)
}

func TestEnsureVisibility(t *testing.T) {
	g := New(nil)

	foo, err := build.ParseBuild("foo/BUILD", []byte(`
go_library(
	name = "foo",
	srcs = ["main.go"],
)
`))
	require.NoError(t, err)

	bar, err := build.ParseBuild("bar/BUILD", []byte(`
go_library(
	name = "bar",
	srcs = ["bar.go"],
	deps = ["//foo"],
)
`))
	require.NoError(t, err)

	g.files["foo"] = foo
	g.files["bar"] = bar

	g.EnsureVisibility("//bar", "//foo")
	g.EnsureVisibility("//bar", "///github.com//foo") // skipped - target in subrepo
	g.EnsureVisibility("//bar", ":foo")               // skipped - local dep
	g.EnsureVisibility("//bar:bar_test", "//bar")     // skipped - also local
	require.Len(t, g.deps, 1)
	require.Equal(t, g.deps[0], &Dependency{
		From: labels.Parse("//bar"),
		To:   labels.Parse("//foo"),
	})

	bs := new(bytes.Buffer)
	err = g.FormatFiles(false, bs)
	require.NoError(t, err)

	fooT := findTargetByName(g.files["foo"], "foo")
	assert.ElementsMatch(t, []string{"//bar:all"}, fooT.AttrStrings("visibility"))

	require.Contains(t, bs.String(), `visibility = ["//bar:all"]`)
}

func TestCheckVisibility(t *testing.T) {
	label := labels.Parse("//foo/bar:baz")
	t.Run("matches exactly", func(t *testing.T) {
		assert.True(t, checkVisibility(label, []string{"//foo/bar:baz"}))
	})
	t.Run("matches all psudo-label", func(t *testing.T) {
		assert.True(t, checkVisibility(label, []string{"//foo/bar:all"}))
	})
	t.Run("matches PUBLIC", func(t *testing.T) {
		assert.True(t, checkVisibility(label, []string{"PUBLIC"}))
	})
	t.Run("matches package wildcard", func(t *testing.T) {
		assert.True(t, checkVisibility(label, []string{"//foo/..."}))
	})

	t.Run("doesnt match a different package wildcard", func(t *testing.T) {
		assert.False(t, checkVisibility(label, []string{"//bar/..."}))
	})
	t.Run("doesnt match a differnt package", func(t *testing.T) {
		assert.False(t, checkVisibility(label, []string{"//bar:all"}))
	})
}
