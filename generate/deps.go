package generate

import (
	"fmt"
	"github.com/please-build/puku/config"
	"github.com/please-build/puku/kinds"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/build"

	"github.com/please-build/puku/knownimports"
)

// resolveImport resolves an import path to a build target. It will return an empty string if the import is for a pkg in
// the go sdk. Otherwise, it will return the build target for that dependency, or an error if it can't be resolved. If
// the target can be resolved to a module that isn't currently added to this project, it will return the build target,
// and record the new module in `u.newModules`. These should later be written to the build graph.
func (u *Update) resolveImport(conf *config.Config, i string) (string, error) {
	if t, ok := u.knownImports[i]; ok {
		return t, nil
	}

	t, err := u.reallyResolveImport(conf, i)
	if err == nil {
		u.knownImports[i] = t
	}
	return t, err
}

// reallyResolveImport actually does the resolution of an import path to a build target.
func (u *Update) reallyResolveImport(conf *config.Config, i string) (string, error) {
	if knownimports.IsInGoRoot(i) {
		return "", nil
	}

	if t := u.installs.Get(i); t != "" {
		return t, nil
	}

	thirdPartyDir := conf.GetThirdPartyDir()

	// Check to see if the target exists in the current repo
	if isInModule(u.conf.ImportPath(), i) || u.conf.ImportPath() == "" {
		t, err := u.localDep(conf, i)
		if err != nil {
			return "", err
		}

		if t != "" {
			return t, nil
		}
		// The above isInModule check only checks the import path. Modules can have import paths that contain the
		// current module, so we should carry on here in case we can resolve this to a third party module
	}

	t := depTarget(u.modules, i, thirdPartyDir)
	if t != "" {
		return t, nil
	}

	// If we're using go_module, we can't automatically add new modules to the graph so we should give up here.
	if u.usingGoModule {
		return "", fmt.Errorf("module not found")
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
	if mod.Module == u.conf.ImportPath() {
		return "", fmt.Errorf("can't find import %q", i)
	}

	u.newModules = append(u.newModules, mod)
	u.modules = append(u.modules, mod.Module)

	// TODO we can probably shortcut this and assume the target is in the above module
	t = depTarget(u.modules, i, thirdPartyDir)
	if t != "" {
		return t, nil
	}

	return "", fmt.Errorf("module not found")
}

// isInScope returns true when the given path is in scope of the current run i.e. if we are going to format the BUILD
// file there.
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

// localDep finds a dependency local to this repository, checking the BUILD file for a go_library target. Returns an
// empty string when no target is found.
func (u *Update) localDep(conf *config.Config, importPath string) (string, error) {
	path := strings.Trim(strings.TrimPrefix(importPath, u.conf.ImportPath()), "/")
	file, err := parseBuildFile(path, u.conf.BuildFileNames())
	if err != nil {
		return "", fmt.Errorf("failed to parse BUILD files in %v: %v", path, err)
	}

	var libTargets []*build.Rule
	for _, rule := range file.Rules("") {
		kind := conf.GetKind(rule.Kind())
		if kind == nil {
			continue
		}

		if kind.Type == kinds.Lib {
			libTargets = append(libTargets, rule)
		}
	}

	// If we can't find the lib target, and the target package is in scope for us to potentially generate it, check if
	// we are going to generate it.
	if len(libTargets) != 0 {
		return buildTarget(libTargets[0].Name(), path, ""), nil
	}

	if !u.isInScope(importPath) {
		return "", fmt.Errorf("resolved %v to a local package, but no library target was found and it's not in scope to generate the target", importPath)
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
			return buildTarget(filepath.Base(importPath), path, ""), nil
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
	bs := new(strings.Builder)
	if subrepo != "" {
		bs.WriteString("///")
		bs.WriteString(subrepo)
	}
	if pkg != "" || subrepo != "" {
		bs.WriteString("//")
	}

	if pkg != "" {
		bs.WriteString(pkg)
		if filepath.Base(pkg) != name {
			bs.WriteString(":")
			bs.WriteString(name)
		}
	} else {
		bs.WriteString(":")
		bs.WriteString(name)
	}
	return bs.String()
}
