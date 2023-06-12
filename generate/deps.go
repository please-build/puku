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
	if strings.HasPrefix(i, u.importPath) || u.importPath == "" {
		t, err := u.localDep(conf, i)
		if err != nil {
			return "", err
		}

		if t != "" {
			return t, nil
		}
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
	path := strings.Trim(strings.TrimPrefix(importPath, u.importPath), "/")
	file, err := parseBuildFile(path, u.buildFileNames)
	if err != nil {
		return "", fmt.Errorf("failed to parse BUILD files in %v: %v", path, err)
	}

	var libTargets []*build.Rule
	for _, rule := range file.Rules("") {
		kind := conf.GetKind(rule.Kind())
		if kind.Type == kinds.Lib {
			libTargets = append(libTargets, rule)
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
				return buildTarget(filepath.Base(importPath), path, ""), nil
			}
		}
		return "", nil
	}

	return buildTarget(libTargets[0].Name(), path, ""), nil
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
