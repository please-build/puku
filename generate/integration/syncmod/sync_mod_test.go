package syncmod

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/generate"
	"github.com/please-build/puku/graph"
	"github.com/please-build/puku/please"
)

func TestModSync(t *testing.T) {
	if err := os.Chdir(os.Getenv("DATA_REPO")); err != nil {
		panic(err)
	}
	conf, err := config.ReadConfig(".")
	require.NoError(t, err)

	conf.PleasePath = filepath.Join(os.Getenv("TMP_DIR"), os.Getenv("DATA_PLZ"))

	plzConf, err := please.QueryConfig(conf.GetPlzPath())
	require.NoError(t, err)

	g := graph.New(plzConf.BuildFileNames())

	u := generate.NewUpdateWithGraph(false, plzConf, g)
	require.NoError(t, err)

	err = u.Sync()
	require.NoError(t, err)

	thirdPartyBuildFile, err := g.LoadFile(conf.GetThirdPartyDir())
	require.NoError(t, err)

	expectedVers := readModFileVersions()

	// We expect to generate the following for the replace in the go.mod:
	// go_mod_download(
	//   name = "github.com_peterebden_buildtools_dl",
	//   module = "github.com/peterebden/buildtools",
	//   version = "v1.6.0",
	// )
	//
	// go_repo(
	//   download = ":github.com_peterebden_buildtools_dl",
	//   module = "github.com/bazelbuild/buildtools",
	// )

	for _, repoRule := range thirdPartyBuildFile.Rules("go_repo") {
		// Check that we've replaced build tools
		if repoRule.AttrString("version") == "" {
			assert.Equal(t, "github.com/bazelbuild/buildtools", repoRule.AttrString("module"))
			assert.Equal(t, ":github.com_peterebden_buildtools_dl", repoRule.AttrString("download"))
			continue
		}
		// All rules start off at v0.0.1 and should be updated to v1.0.0 as per the go.mod
		assert.Equal(t, expectedVers[repoRule.AttrString("module")], repoRule.AttrString("version"))
	}

	dlRules := thirdPartyBuildFile.Rules("go_mod_download")
	require.Len(t, dlRules, 1)

	dlRule := dlRules[0]
	assert.Equal(t, "github.com_peterebden_buildtools_dl", dlRule.Name())
	assert.Equal(t, "github.com/peterebden/buildtools", dlRule.AttrString("module"))
	assert.Equal(t, expectedVers[dlRule.AttrString("module")], dlRule.AttrString("version"))
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
