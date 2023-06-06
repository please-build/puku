package generate

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (u *Update) localDep(importPath string) (string, error) {
	path := strings.Trim(strings.TrimPrefix(importPath, u.importPath), "/")
	file, err := parseBuildFile(path, u.buildFileNames)
	if err != nil {
		return "", err
	}
	libTargets := file.Rules("go_library")
	if len(libTargets) == 0 {
		// TODO(#3): we should check if 1) the target package is in scope for us to visit it, and 2) that doing so would
		// 	generate this library target. We can return the label that it would generate.
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
