package syncmod

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/please-build/buildtools/build"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/graph"
	"github.com/please-build/puku/please"
	"github.com/please-build/puku/sync"
)

func TestModSync(t *testing.T) {
	if err := os.Chdir(os.Getenv("DATA_REPO")); err != nil {
		panic(err)
	}

	// Setup puku and please config
	conf, err := config.ReadConfig(".")
	require.NoError(t, err)

	conf.PleasePath = filepath.Join(os.Getenv("TMP_DIR"), os.Getenv("DATA_PLZ"))

	plzConf, err := please.QueryConfig(conf.GetPlzPath())
	require.NoError(t, err)

	// Parse the puku graph of test repo build files
	g := graph.New(plzConf.BuildFileNames())
	err = sync.SyncToStdout("text", plzConf, g)
	require.NoError(t, err)

	// Fetch the generated third_party/go build file
	thirdPartyBuildFile, err := g.LoadFile(conf.GetThirdPartyDir())
	require.NoError(t, err)

	// Read version info from the go.mod file
	expectedVers := readModFileVersions()

	// We expect to generate the following for the replace directives in the go.mod:
	// go_mod_download(
	//   name = "github.com_peterebden_buildtools_dl",
	//   module = "github.com/peterebden/buildtools",
	//   version = "v1.6.0",
	// )
	//
	// go_repo(
	//   download = ":github.com_peterebden_buildtools_dl",
	//   module = "github.com/bazelbuild/buildtools",
	//   labels = ["go_replace_directive"],
	// )
	//
	// go_repo(
	//   module = "github.com/stretchr/testify",
	//   version = "v1.3.0",
	//   labels = ["go_replace_directive"],
	// )

	for _, repoRule := range thirdPartyBuildFile.Rules("go_repo") {
		t.Run(repoRule.AttrString("module"), func(t *testing.T) {
			// Check that we've replaced build tools
			if repoRule.AttrString("module") == "github.com/bazelbuild/buildtools" {
				// Assert there's no version set
				assert.Equal(t, "", repoRule.AttrString("version"))
				// Ensure the download attribute is set
				assert.Equal(t, ":github.com_peterebden_buildtools_dl", repoRule.AttrString("download"))
				// Check that a label has been added
				labels := listLabels(repoRule)
				assert.Contains(t, labels, "go_replace_directive")
				return
			}

			// Check that testify is the only other one labelled for a replace directive
			labels := listLabels(repoRule)
			if repoRule.AttrString("module") == "github.com/stretchr/testify" {
				assert.Contains(t, labels, "go_replace_directive")
			} else {
				assert.NotContains(t, labels, "go_replace_directive")
				// Ensure there aren't empty labels list attributes
				if len(labels) == 0 {
					assert.Nil(t, repoRule.Attr("labels"))
				}
			}

			// All rules start off at v0.0.1 and should be updated to their version listed in the go.mod
			assert.Equal(t, expectedVers[repoRule.AttrString("module")], repoRule.AttrString("version"))
		})
	}

	dlRules := thirdPartyBuildFile.Rules("go_mod_download")
	require.Len(t, dlRules, 1)

	dlRule := dlRules[0]
	assert.Equal(t, "github.com_peterebden_buildtools_dl", dlRule.Name())
	assert.Equal(t, "github.com/peterebden/buildtools", dlRule.AttrString("module"))
	assert.Equal(t, expectedVers[dlRule.AttrString("module")], dlRule.AttrString("version"))
}

func listLabels(rule *build.Rule) []string {
	labelsExpr := rule.Attr("labels")
	if labelsExpr == nil {
		return []string{}
	}
	labels := make([]string, 0, len(labelsExpr.(*build.ListExpr).List))
	for _, labelExpr := range labelsExpr.(*build.ListExpr).List {
		labels = append(labels, labelExpr.(*build.StringExpr).Value)
	}
	return labels
}

func readModFileVersions() map[string]string {
	f, err := os.ReadFile("go.mod")
	if err != nil {
		panic(err)
	}

	file, err := modfile.Parse("go.mod", f, nil)
	if err != nil {
		panic(err)
	}

	ret := make(map[string]string)
	for _, req := range file.Require {
		ret[req.Mod.Path] = req.Mod.Version
	}

	for _, replace := range file.Replace {
		ret[replace.New.Path] = replace.New.Version
	}
	return ret
}
