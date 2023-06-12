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
	if isInModule(u.importPath, i) || u.importPath == "" {
		t, err := u.localDep(i)
		if err != nil {
			return "", err
		}

		if t != "" {
			return t, nil
		}
		// The above isInModule check only checks the import path. Modules can have import paths that contain the
		// current module, so we should carry on here in case we can resolve this to a third party module
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

	// If the package belongs to this module, we should have found this package when resolving local imports above. We
	// don't want to resolve this like a third party module, so we should return an error here.
	if mod.Module == u.importPath {
		return "", fmt.Errorf("can't find import %q", i)
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
	if len(libTargets) != 0 {
		return "//" + path + ":" + libTargets[0].Name(), nil
	}
	if !u.isInScope(importPath) {
		return "", nil
	}

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

func depTarget(modules []string, importPath, thirdPartyFolder string) string {
	module := moduleForPackage(modules, importPath)
	if module == "" {
		// If we can't find this import, we can return nothing and the build rule will fail at build time reporting a
		// sensible error. It may also be an import from the go SDK which is fine.
		return ""
	}

	subrepoName := subrepoName(module, thirdPartyFolder)
	packageName := strings.TrimPrefix(strings.TrimPrefix(importPath, module), "/")
	name := filepath.Base(packageName)
	if packageName == "" {
		name = filepath.Base(module)
	}

	return buildTarget(name, packageName, subrepoName)
}

// isInModule checks to see if the given import path is in the provided module. This check is based entirely off the
// paths, so doesn't actually check if the package exists.
func isInModule(module, path string) bool {
	pathParts := strings.Split(path, "/")
	moduleParts := strings.Split(module, "/")
	if len(moduleParts) > len(pathParts) {
		return false
	}

	for i := range moduleParts {
		if pathParts[i] != moduleParts[i] {
			return false
		}
	}
	return true
}

func moduleForPackage(modules []string, importPath string) string {
	module := ""
	for _, mod := range modules {
		ok := isInModule(mod, importPath)
		if ok && len(mod) > len(module) {
			module = mod
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
