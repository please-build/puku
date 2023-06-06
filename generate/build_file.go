package generate

import (
	"fmt"
	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/edit"
	"os"
	"path/filepath"
)

func saveAndFormatBuildFile(buildFile *build.File) error {
	f, err := os.Create(buildFile.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(build.Format(buildFile))
	return err
}

// parseBuildFile loops through the available build file names to create a new build file or open the existing
// one.
func parseBuildFile(path string, fileNames []string) (*build.File, error) {
	for _, name := range fileNames {
		filePath := filepath.Join(path, name)
		if f, err := os.Lstat(filePath); os.IsNotExist(err) {
			return build.ParseBuild(filePath, nil)
		} else if !f.IsDir() {
			bs, err := os.ReadFile(filePath)
			if err != nil {
				return nil, err
			}
			return build.ParseBuild(filePath, bs)
		}
	}
	return nil, fmt.Errorf("folders exist with the build file names in directory %v %v", path, fileNames)
}

func newRuleExpr(kind, name string) *build.Rule {
	rule, _ := edit.ExprToRule(&build.CallExpr{
		X:    &build.Ident{Name: kind},
		List: []build.Expr{},
	}, kind)

	rule.SetAttr("name", NewStringExpr(name))

	return rule
}
