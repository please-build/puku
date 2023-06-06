package generate

import (
	"fmt"
	"github.com/bazelbuild/buildtools/build"
	"os"
	"path/filepath"
	"strings"
)

func (u *Update) isInScope(path string) bool {
	for _, p := range u.paths {
		if p == path {
			return true
		}
		if strings.HasSuffix(p, "...") {
			p = filepath.Clean(strings.TrimSuffix(p, "..."))
			if strings.HasPrefix(path, p) || p == "." {
				return true
			}
		}
	}
	return false
}

func (u *Update) localDep(importPath string) (string, error) {
	path := strings.Trim(strings.TrimPrefix(importPath, u.importPath), "/")
	file, err := parseBuildFile(path, u.buildFileNames)
	if err != nil {
		return "", fmt.Errorf("failed to parse BUILD files in %v: %v", path, err)
	}

	var libTargets []*build.Rule
	for kind, kindType := range u.kinds {
		if kindType == KindType_Lib {
			libTargets = append(libTargets, file.Rules(kind)...)
		}
	}

	// If we can't find the lib target, and the target package is in scope for us to potentially generate it, check if
	// we are going to generate it.
	if len(libTargets) == 0 && u.isInScope(path) {
		files, err := ImportDir(path)
		if err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", fmt.Errorf("failed to import %v: %v", path, err)
		}

		// If there are any non-test sources, then we will generate a go_library here later on. Return that target name.
		for _, f := range files {
			if !f.IsTest() {
				return fmt.Sprintf("//%v:%v", path, filepath.Base(importPath)), nil
			}
		}
		return "", nil
	}

	return "//" + path + ":" + libTargets[0].Name(), nil
}

func depTarget(modules []string, importPath, thirdPartyFolder string) string {
	module := moduleForPackage(modules, importPath)
	if module == "" {
		// If we can't find this import, we can return nothing and the build rule will fail at build time reporting a
		// sensible error. It may also be an import from the go SDK which is fine.
		return ""
	}

	subrepoName := subrepoName(module, thirdPartyFolder)
	packageName := strings.TrimPrefix(importPath, module)
	packageName = strings.TrimPrefix(packageName, "/")
	name := filepath.Base(packageName)
	if packageName == "" {
		name = filepath.Base(module)
	}

	return buildTarget(name, packageName, subrepoName)
}

func moduleForPackage(modules []string, importPath string) string {
	module := ""
	for _, mod := range modules {
		if strings.HasPrefix(importPath, mod) {
			if len(module) < len(mod) {
				module = mod
			}
		}
	}
	return module
}

func subrepoName(module, thirdPartyFolder string) string {
	return filepath.Join(thirdPartyFolder, strings.ReplaceAll(module, "/", "_"))
}

func buildTarget(name, pkg, subrepo string) string {
	if name == "" {
		name = filepath.Base(pkg)
	}
	target := fmt.Sprintf("%v:%v", pkg, name)
	if subrepo == "" {
		return fmt.Sprintf("//%v", target)
	}
	return fmt.Sprintf("///%v//%v", subrepo, target)
}
