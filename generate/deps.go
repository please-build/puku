package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/please-build/buildtools/build"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/fs"
	"github.com/please-build/puku/kinds"
	"github.com/please-build/puku/knownimports"
)

// resolveImport resolves an import path to a build target. It will return an empty string if the import is for a pkg in
// the go sdk. Otherwise, it will return the build target for that dependency, or an error if it can't be resolved. If
// the target can be resolved to a module that isn't currently added to this project, it will return the build target,
// and record the new module in `u.newModules`. These should later be written to the build graph.
func (u *Update) resolveImport(conf *config.Config, i string) (string, error) {
	if t, ok := u.resolvedImports[i]; ok {
		return t, nil
	}

	if t := conf.GetKnownTarget(i); t != "" {
		return t, nil
	}

	t, err := u.reallyResolveImport(conf, i)
	if err == nil {
		u.resolvedImports[i] = t
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
	if fs.IsSubdir(u.plzConf.ImportPath(), i) || u.plzConf.ImportPath() == "" {
		t, err := u.localDep(i)
		if err != nil {
			return "", err
		}

		if t != "" {
			return t, nil
		}
		// The above isSubdir check only checks the import path. Modules can have import paths that contain the
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

	log.Infof("Resolving module for %v...", i)

	// Otherwise try and resolve it to a new dep via the module proxy. We assume the module will contain the package.
	// Please will error out in a reasonable way if it doesn't.
	// TODO it would be more correct to download the module and check it actually contains the package
	mod, err := u.proxy.ResolveModuleForPackage(i)
	if err != nil {
		return "", err
	}

	log.Infof("Resolved to %v... done", mod.Module)

	// If the package belongs to this module, we should have found this package when resolving local imports above. We
	// don't want to resolve this like a third party module, so we should return an error here.
	if mod.Module == u.plzConf.ImportPath() {
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
	}
	return false
}

// localDep finds a dependency local to this repository, checking the BUILD file for a go_library target. Returns an
// empty string when no target is found.
func (u *Update) localDep(importPath string) (string, error) {
	path := strings.Trim(strings.TrimPrefix(importPath, u.plzConf.ImportPath()), "/")
	// If we're using GOPATH based resolution, we don't have a prefix to base whether a path is package local or not. In
	// this case, we need to check if the directory exists. If it doesn't it's not a local import.
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return "", nil
	}
	file, err := u.graph.LoadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to parse BUILD files in %v: %v", path, err)
	}

	conf, err := config.ReadConfig(path)
	if err != nil {
		return "", err
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
		return BuildTarget(libTargets[0].Name(), path, ""), nil
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
			return BuildTarget(filepath.Base(importPath), path, ""), nil
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

	packageName := strings.TrimPrefix(strings.TrimPrefix(importPath, module), "/")
	return SubrepoTarget(module, thirdPartyFolder, packageName)
}

func moduleForPackage(modules []string, importPath string) string {
	module := ""
	for _, mod := range modules {
		ok := fs.IsSubdir(mod, importPath)
		if ok && len(mod) > len(module) {
			module = mod
		}
	}
	return module
}

func SubrepoTarget(module, thirdPartyFolder, packageName string) string {
	subrepoName := SubrepoName(module, thirdPartyFolder)

	name := filepath.Base(packageName)
	if packageName == "" {
		name = filepath.Base(module)
	}

	return BuildTarget(name, packageName, subrepoName)
}

func SubrepoName(module, thirdPartyFolder string) string {
	return filepath.Join(thirdPartyFolder, strings.ReplaceAll(module, "/", "_"))
}

func BuildTarget(name, pkgDir, subrepo string) string {
	bs := new(strings.Builder)
	if subrepo != "" {
		bs.WriteString("///")
		bs.WriteString(subrepo)
	}

	if pkgDir != "" || subrepo != "" {
		bs.WriteString("//")
	}

	if pkgDir == "." {
		pkgDir = ""
	}

	if pkgDir != "" {
		bs.WriteString(pkgDir)
		if filepath.Base(pkgDir) != name {
			bs.WriteString(":")
			bs.WriteString(name)
		}
	} else {
		bs.WriteString(":")
		bs.WriteString(name)
	}
	return bs.String()
}
