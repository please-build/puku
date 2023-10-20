package migrate

import (
	"github.com/bazelbuild/buildtools/build"
	"github.com/please-build/puku/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMigrateGoModule(t *testing.T) {
	m := &Migrate{
		graph:            graph.New([]string{"BUILD"}),
		thirdPartyFolder: "third_party/go",
		moduleRules:      map[string]*moduleParts{},
	}

	thirdPartyFile, err := build.ParseBuild("third_party/go", []byte(`
go_module(
	name = "test",
	module = "github.com/example/example",
	version = "v1.0.0",
	install = ["..."],
)
	`))
	if err != nil {
		panic(err)
	}
	m.graph.SetFile("third_party/go", thirdPartyFile)

	err = m.Migrate(false, "third_party/go")
	require.NoError(t, err)

	rule := graph.FindTargetByName(thirdPartyFile, "test")
	require.NotNil(t, rule)

	assert.Equal(t, "v1.0.0", rule.AttrString("version"))
	assert.Equal(t, "github.com/example/example", rule.AttrString("module"))
	assert.ElementsMatch(t, []string{"..."}, rule.AttrStrings("install"))
}

func TestMigrateGoModuleWithParts(t *testing.T) {
	m := &Migrate{
		graph:            graph.New([]string{"BUILD"}),
		thirdPartyFolder: "third_party/go",
		moduleRules:      map[string]*moduleParts{},
	}

	thirdPartyFile, err := build.ParseBuild("third_party/go", []byte(`
go_mod_download(
	name = "test_dl",
	module = "github.com/example/example",
	version = "v1.0.0",
)

go_module(
	name = "test_foo",
	download = ":test_dl",
	module = "github.com/example/example",
	install = ["foo/..."],
)

go_module(
	name = "test_bar",
	download = ":test_dl",
	module = "github.com/example/example",
	install = ["bar/..."],
)

go_module(
	name = "test_bin_install",
	download = ":test_dl",
	module = "github.com/example/example",
	install = ["cmd"],
	binary = True,
)

go_module(
	name = "test_bin",
	download = ":test_dl",
	module = "github.com/example/example",
	binary = True,
)

go_module(
	name = "test_bin_only",
	version = "v1.0.0",
	module = "github.com/example/bin-only",
	binary = True,
)
	`))
	if err != nil {
		panic(err)
	}
	m.graph.SetFile("third_party/go", thirdPartyFile)

	err = m.Migrate(false, "third_party/go")
	require.NoError(t, err)

	repoRules := thirdPartyFile.Rules("go_repo")
	require.Len(t, repoRules, 2)
	repoRule := graph.FindTargetByName(thirdPartyFile, "github.com_example_example")

	assert.Equal(t, "v1.0.0", repoRule.AttrString("version"))
	assert.Equal(t, "github.com/example/example", repoRule.AttrString("module"))
	assert.ElementsMatch(t, []string{"foo/...", "bar/..."}, repoRule.AttrStrings("install"))

	fooAlias := graph.FindTargetByName(thirdPartyFile, "test_foo")
	assert.Equal(t, []string{"///third_party/go/github.com_example_example//:installs"}, fooAlias.AttrStrings("exported_deps"))

	barAlias := graph.FindTargetByName(thirdPartyFile, "test_bar")
	assert.Equal(t, []string{"///third_party/go/github.com_example_example//:installs"}, barAlias.AttrStrings("exported_deps"))

	binWithInstallAlias := graph.FindTargetByName(thirdPartyFile, "test_bin_install")
	assert.Equal(t, []string{"///third_party/go/github.com_example_example//cmd"}, binWithInstallAlias.AttrStrings("exported_deps"))

	binNoInstallAlias := graph.FindTargetByName(thirdPartyFile, "test_bin")
	assert.Equal(t, []string{"///third_party/go/github.com_example_example//:example"}, binNoInstallAlias.AttrStrings("exported_deps"))

	binOnlyAlias := graph.FindTargetByName(thirdPartyFile, "test_bin_only")
	assert.Equal(t, []string{"///third_party/go/github.com_example_bin-only//:bin-only"}, binOnlyAlias.AttrStrings("exported_deps"))

}

func TestModuleAlias(t *testing.T) {
	m := &Migrate{
		graph:            graph.New([]string{"BUILD"}),
		thirdPartyFolder: "third_party/go",
		moduleRules:      map[string]*moduleParts{},
	}

	thirdPartyFile, err := build.ParseBuild("third_party/go", []byte(`
go_mod_download(
	name = "test_dl",
	module = "github.com/fork/example",
	version = "v1.0.0",
)

go_module(
	name = "test",
	download = ":test_dl",
	module = "github.com/example/example",
)

	`))
	if err != nil {
		panic(err)
	}
	m.graph.SetFile("third_party/go", thirdPartyFile)

	err = m.Migrate(false, "third_party/go")
	require.NoError(t, err)

	repoRule := graph.FindTargetByName(thirdPartyFile, "test")
	require.NotNil(t, repoRule)

	assert.Equal(t, "github.com/example/example", repoRule.AttrString("module"))
	assert.Equal(t, ":test_dl", repoRule.AttrString("download"))
	assert.ElementsMatch(t, []string{"."}, repoRule.AttrStrings("install"))

	assert.NotNil(t, graph.FindTargetByName(thirdPartyFile, "test_dl"))
}

func TestAliasesInOtherDirs(t *testing.T) {
	m := &Migrate{
		graph:            graph.New([]string{"BUILD"}),
		thirdPartyFolder: "third_party/go",
		moduleRules:      map[string]*moduleParts{},
	}

	thirdPartyK8sFile, err := build.ParseBuild("third_party/go/kubernetes", []byte(`
go_module(
    name = "api",
    install = ["..."],
    module = "k8s.io/api",
    version = "v0.24.17",
)
	`))
	if err != nil {
		panic(err)
	}
	m.graph.SetFile("third_party/go/kubernetes", thirdPartyK8sFile)

	thirdPartyFile, err := build.ParseBuild("third_party/go", nil)
	if err != nil {
		panic(err)
	}
	m.graph.SetFile("third_party/go", thirdPartyFile)

	err = m.Migrate(false, "third_party/go", "third_party/go/kubernetes")
	require.NoError(t, err)

	repoRule := graph.FindTargetByName(thirdPartyFile, "api")
	require.NotNil(t, repoRule)

	aliasRule := graph.FindTargetByName(thirdPartyK8sFile, "api")
	require.NotNil(t, aliasRule)

	assert.ElementsMatch(t, []string{"///third_party/go/k8s.io_api//:installs"}, aliasRule.AttrStrings("exported_deps"))
}
