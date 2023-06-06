package generate

import (
	"fmt"
	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/edit"
	"os"
	"path/filepath"
)

func saveAndFormatBuildFile(buildFile *build.File) error {
	if len(buildFile.Stmt) == 0 {
		return nil
	}

	f, err := os.Create(buildFile.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(build.Format(buildFile))
	return err
}

// parseBuildFile tries to find and parse a build file in a directory. If it finds one, it will return true and the
// parsed build file. If it doesn't find one, it returns false, and an empty build file.
func parseBuildFile(path string, fileNames []string) (*build.File, error) {
	validFilename := ""
	for _, name := range fileNames {
		filePath := filepath.Join(path, name)
		if f, err := os.Lstat(filePath); os.IsNotExist(err) {
			// This file name is available. Use the first one we find in the list.
			if validFilename == "" {
				validFilename = filePath
			}
		} else if !f.IsDir() { // this is a common issue on macos where paths are case insensitive...
			bs, err := os.ReadFile(filePath)
			if err != nil {
				return nil, err
			}
			file, err := build.ParseBuild(filePath, bs)
			return file, err
		}
	}
	if validFilename == "" {
		return nil, fmt.Errorf("folders exist with the build file names in directory %v %v", path, fileNames)
	}

	// Otherwise returns a new empty file. We didn't find one.
	return build.ParseBuild(validFilename, nil)
}

func newRuleExpr(kind, name string) *build.Rule {
	rule, _ := edit.ExprToRule(&build.CallExpr{
		X:    &build.Ident{Name: kind},
		List: []build.Expr{},
	}, kind)

	rule.SetAttr("name", NewStringExpr(name))

	return rule
}
