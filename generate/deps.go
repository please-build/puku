package generate

import (
	"fmt"
	"path/filepath"
	"strings"
)

func depTarget(modules []string, thisModule, importPath, thirdPartyFolder string) string {
	module := moduleForPackage(append(modules, thisModule), importPath)
	if module == "" {
		// If we can't find this import, we can return nothing and the build rule will fail at build time reporting a
		// sensible error. It may also be an import from the go SDK which is fine.
		return ""
	}

	subrepoName := subrepoName(module, thisModule, thirdPartyFolder)
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

func subrepoName(module, thisModule, thirdPartyFolder string) string {
	if thisModule == module {
		return ""
	}
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
