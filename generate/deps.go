package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/build"

	"github.com/please-build/puku/knownimports"
)

func (u *Update) resolveImport(i string) (string, error) {
	if t, ok := u.knownImports[i]; ok {
		return t, nil
	}

	t, err := u.reallyResolveImport(i)
	if err == nil {
		u.knownImports[i] = t
	}
	return t, err
}

func (u *Update) reallyResolveImport(i string) (string, error) {
	if knownimports.IsInGoRoot(i) {
		return "", nil
	}

	if t := u.installs.Get(i); t != "" {
		return t, nil
	}

	// Check to see if the target exists in the current repo
	if strings.HasPrefix(i, u.importPath) || u.importPath == "" {
		t, err := u.localDep(i)
		if err != nil {
			return "", err
		}

		if t != "" {
			return t, nil
		}
	}

	t := depTarget(u.modules, i, u.thirdPartyDir)
	if t != "" {
		return t, nil
	}

	// Otherwise try and resolve it to a new dep via the module proxy. We assume the module will contain the package.
	// Please will error out in a reasonable way if it doesn't.
	// TODO it would be more correct to download the module and check it actually contains the package
	mod, err := u.proxy.ResolveModuleForPackage(i)
	if err != nil {
		return "", err
	}
	u.newModules = append(u.newModules, mod)
	u.modules = append(u.modules, mod.Module)

	t = depTarget(u.modules, i, u.thirdPartyDir)
	if t != "" {
		return t, nil
	}
	return "", fmt.Errorf("module not found")
}

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
		if kindType == KindTypeLib {
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

func localTarget(name string) string {
	return ":" + name
}
