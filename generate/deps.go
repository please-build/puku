package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/please-build/buildtools/build"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/fs"
	"github.com/please-build/puku/kinds"
	"github.com/please-build/puku/knownimports"
)

// resolveImport resolves an import path to a build target. It will return an empty string if the import is for a pkg in
// the go sdk. Otherwise, it will return the build target for that dependency, or an error if it can't be resolved. If
// the target can be resolved to a module that isn't currently added to this project, it will return the build target,
// and record the new module in `u.newModules`. These should later be written to the build graph.
func (u *updater) resolveImport(conf *config.Config, i string) (string, error) {
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
func (u *updater) reallyResolveImport(conf *config.Config, i string) (string, error) {
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

// resolveTSImport resolves an import path to a build target. It will return an
// empty string if the import is for a third party package. Otherwise, it will
// return the build target for that dependency, or an error if it can't be resolved.
func (u *updater) resolveTSImport(conf *config.Config, tsConfig *config.TSConfig, f *SourceFile, importPath string, currentRule *edit.Rule) (string, error) {
	var t string
	var err error

	var importPaths []string
	type tsAlias struct {
		alias   string
		targets []string
	}
	var matchedAlias *tsAlias

	tsPaths := make(map[string][]string)
	if tsConfig != nil {
		tsPaths = tsConfig.CompilerOptions.Paths
	}

	for alias, targets := range tsPaths {
		re := regexp.MustCompile(wildCardToRegexp(alias))
		matched := re.FindString(importPath)
		if err != nil {
			return t, err
		}

		if matched != "" {
			matchedAlias = &tsAlias{
				alias:   alias,
				targets: targets,
			}
		}
	}

	// Ignore all imports that aren't relative
	if !strings.HasPrefix(importPath, ".") && matchedAlias == nil {
		log.Debugf("Skipping TS import (not relative): %s", importPath)
		return t, nil
	}

	if matchedAlias != nil {
		// TODO: at the moment we only support the first target and we only support
		// aliases that are simple prefix replacements
		target := matchedAlias.targets[0]
		alias := matchedAlias.alias
		origImportPath := importPath

		aliasPrefix := alias[0:strings.Index(alias, "*")]
		targetPrefix := target[0:strings.Index(target, "*")]

		importPath = strings.Replace(importPath, aliasPrefix, targetPrefix, 1)
		importPath = filepath.Join(tsConfig.Dir, importPath)
		log.Debugf("alias matched %s %s; newImportPath: %s", alias, origImportPath, importPath)
	}

	// If importPath is a folder then append `index.{ts,tsx}`
	if filepath.Ext(importPath) == "" {
		// filepath.Join removes the './' at the beginning of the import so we can't
		// use it
		importPaths = append(importPaths, importPath+".ts")
		importPaths = append(importPaths, importPath+".tsx")
		importPaths = append(importPaths, importPath+string(filepath.Separator)+"index.ts")
		importPaths = append(importPaths, importPath+string(filepath.Separator)+"index.tsx")
		log.Debugf("adding file extensions and index paths: %s", importPaths)
	} else {
		importPaths = append(importPaths, importPath)
	}

	// Try every possible import path
	for _, path := range importPaths {
		fullPath := path
		if strings.HasPrefix(path, ".") {
			fullPath = filepath.Join(f.Dir(), path)
		}
		log.Debugf("fullPath: %s", fullPath)

		if t, ok := u.resolvedImports[fullPath]; ok {
			return t, nil
		}

		// TODO
		if t := conf.GetKnownTarget(fullPath); t != "" {
			return t, nil
		}

		// Check to see if the target exists in the current repo
		t, err = u.localTSDep(fullPath, currentRule)
		if err != nil {
			return "", err
		}

		if t != "" {
			u.resolvedImports[fullPath] = t
			return t, nil
		}
	}

	return "", nil
}

func wildCardToRegexp(pattern string) string {
	components := strings.Split(pattern, "*")
	if len(components) == 1 {
		// if len is 1, there are no *'s, return exact match pattern
		return "^" + pattern + "$"
	}
	var result strings.Builder
	for i, literal := range components {

		// Replace * with .*
		if i > 0 {
			result.WriteString("(.*)")
		}

		// Quote any regular expression meta characters in the
		// literal text.
		result.WriteString(regexp.QuoteMeta(literal))
	}
	return "^" + result.String() + "$"
}

// isInScope returns true when the given path is in scope of the current run i.e. if we are going to format the BUILD
// file there.
func (u *updater) isInScope(path string) bool {
	for _, p := range u.paths {
		if p == path {
			return true
		}
	}
	return false
}

// localDep finds a dependency local to this repository, checking the BUILD file for a go_library target. Returns an
// empty string when no target is found.
func (u *updater) localDep(importPath string) (string, error) {
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
		return edit.BuildTarget(libTargets[0].Name(), path, ""), nil
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
			return edit.BuildTarget(filepath.Base(importPath), path, ""), nil
		}
	}
	return "", nil
}

// localTSDep finds a dependency local to this repository, checking the BUILD
// file for a js_library target. Returns an empty string when no target is found.
func (u *updater) localTSDep(importPath string, currentRule *edit.Rule) (string, error) {
	path := filepath.Dir(importPath)
	// Check the directory exists. If it doesn't it's not a local import.
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		log.Debugf("dir doesn't exist %s", path)
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

	// TODO allow other rule names?
	for _, rule := range file.Rules("js_library") {
		kind := conf.GetKind(rule.Kind())
		if kind == nil {
			continue
		}

		// Skip rules that are the same as the current rule to prevent circular
		// imports
		if currentRule.Dir == path && rule.Name() == currentRule.Name() {
			continue
		}

		if kind.Type == kinds.Lib {
			ruleSrcs, err := u.eval.EvalGlobs(path, rule, kind.SrcsAttr)
			if err != nil {
				return "", err
			}

			// TODO if rule is the same as the current rule then skip

			// Check if import file matches any of the srcs
			for _, src := range ruleSrcs {
				fileName := filepath.Base(importPath)
				// Files don't have to have an extension. If they don't then they could
				// map with .ts or .tsx files.
				if src == fileName || src == fileName+".ts" || src == fileName+".tsx" {
					log.Debugf("found rule for import %s: %s:%s", importPath, path, rule.Name())
					return edit.BuildTarget(rule.Name(), path, ""), nil
				}
			}
		}
	}

	// if !u.isInScope(importPath) {
	// 	return "", fmt.Errorf("resolved %v to a local package, but no library target was found and it's not in scope to generate the target", importPath)
	// }

	// files, err := ImportDir(path)
	// if err != nil {
	// 	if os.IsNotExist(err) {
	// 		return "", nil
	// 	}
	// 	return "", fmt.Errorf("failed to import %v: %v", path, err)
	// }

	// If there are any non-test sources, then we will generate a js_library here later on. Return that target name.
	// for _, f := range files {
	// 	if !f.IsTest() {
	// 		return BuildTarget(filepath.Base(importPath), path, ""), nil
	// 	}
	// }

	log.Debugf("failed to find rule for import: %s", importPath)
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
	return edit.SubrepoTarget(module, thirdPartyFolder, packageName)
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
