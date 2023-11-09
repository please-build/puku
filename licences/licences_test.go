package licences

import (
	"github.com/bazelbuild/buildtools/build"
	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/graph"
	"github.com/please-build/puku/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetLicences(t *testing.T) {
	p := proxy.New(proxy.DefaultURL)
	l := Licenses{
		proxy: p,
	}

	ls, err := l.Get("github.com/stretchr/testify", "v1.8.4")
	require.NoError(t, err)

	require.Len(t, ls, 1)
	assert.Equal(t, ls[0], "MIT")
}

func TestUpdateLicences(t *testing.T) {
	g := graph.New([]string{"BUILD_FILE"})
	fileContent := `
go_module(
	name = "testify",
	module = "github.com/stretchr/testify",
	version = "v1.8.4",
)
go_repo(
	name = "protobuf",
	module = "github.com/protocolbuffers/protobuf",
	version = "v3.19.6+incompatible",
)
`
	thirdPartFile, err := build.ParseBuild("third_party/go", []byte(fileContent))
	require.NoError(t, err)

	g.SetFile("third_party/go", thirdPartFile)
	p := proxy.New(proxy.DefaultURL)
	l := Licenses{
		proxy: p,
		graph: g,
	}

	err = l.Update([]string{"third_party/go"}, false)
	require.NoError(t, err)

	testify := edit.FindTargetByName(thirdPartFile, "testify")
	require.NotNil(t, testify)
	assert.ElementsMatch(t, []string{"MIT"}, testify.AttrStrings("licences"))

	protobuf := edit.FindTargetByName(thirdPartFile, "protobuf")
	require.NotNil(t, protobuf)
	assert.ElementsMatch(t, []string{"BSD-3-Clause"}, protobuf.AttrStrings("licences"))
}
