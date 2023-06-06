package generate

import (
	"errors"
	"fmt"
	"github.com/please-build/paku/knownimports"
	"github.com/please-build/paku/proxy"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/build"
)

type Update struct {
	plzPath, thirdPartyDir string
	buildFileNames         []string
	importPath             string
	modules                []string
	proxy                  *proxy.Proxy

	newModules []*proxy.Module
}

func NewUpdate(plzPath, thirdPartyDir string, buildFileNames []string) *Update {
	return &Update{
		plzPath:        plzPath,
		thirdPartyDir:  thirdPartyDir,
		buildFileNames: buildFileNames,
		proxy:          proxy.New("https://proxy.golang.org"),
	}
}

// Update updates an existing Please project. It may create new BUILD files, however it tries to respect existing build
// rules, updating them as appropriate.
func (u *Update) Update(paths []string) error {
	var err error

	u.importPath, err = u.getImportPath()
	if err != nil {
		return err
	}

	u.modules, err = u.getModules()
	if err != nil {
		return err
	}

	return u.update(paths)
}

// getImportPath returns the configured import path of this please project
func (u *Update) getImportPath() (string, error) {
	cmd := exec.Command(u.plzPath, "query", "config", "plugin.go.importpath")
	cmd.Stderr = os.Stderr
	importPath, err := cmd.Output() //TODO this is a naff
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(importPath)), nil
}

// getModules returns the defined third party modules in this project
func (u *Update) getModules() ([]string, error) {
	cmd := exec.Command(u.plzPath, "query", "print", "--label=go_module:", fmt.Sprintf("//%v/...", u.thirdPartyDir))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output() //TODO this is a naff
	if err != nil {
		return nil, err
	}

	modVerPairs := strings.Split(strings.TrimSpace(string(out)), "\n")

	ret := make([]string, 0, len(modVerPairs))
	for _, m := range modVerPairs {
		parts := strings.Split(m, "@")
		ret = append(ret, parts[0])
	}

	return ret, nil
}

// update loops through the provided paths, updating and creating any build rules it finds.
func (u *Update) update(paths []string) error {
	// TODO handle when the rule kind changes because we added some cgo sources

	for _, path := range paths {
		// Find all the files in the dir
		sources, err := ImportDir(path)
		if err != nil {
			return err
		}

		// Parse the build file
		file, err := parseBuildFile(path, u.buildFileNames)
		if err != nil {
			return err
		}

		// Read existing rules from file
		rules, calls := readRulesFromFile(file)

		// Allocate the sources to the rules, creating new rules as necessary
		newRules, err := allocateSources(path, sources, unallocatedSources(sources, rules), rules)
		if err != nil {
			return err
		}

		rules = append(rules, newRules...)

		// Update the existing call expressions in the build file
		if err := u.updateCalls(calls, rules, sources); err != nil {
			return err
		}

		// Create new call expressions for the new rules
		for _, r := range newRules {
			rule := newRuleExpr("", "")
			if err := u.updateDeps(r, rules, sources); err != nil {
				return err
			}
			populateRule(rule, r)
			file.Stmt = append(file.Stmt, rule.Call)
		}

		// Save the file back
		if err := saveAndFormatBuildFile(file); err != nil {
			return err
		}
	}

	// Save any new modules we needed back to the third party file
	return u.addNewModules()
}

func (u *Update) addNewModules() error {
	file, err := parseBuildFile(u.thirdPartyDir, u.buildFileNames)
	if err != nil {
		return err
	}

	var mods []*proxy.Module
	existingRules := make(map[string]*build.Rule)
	for _, rule := range file.Rules("go_repo") {
		// TODO handle when the version is specified by a download rule
		mod, ver := rule.AttrString("module"), rule.AttrString("version")
		existingRules[rule.AttrString("module")] = rule
		mods = append(mods, &proxy.Module{Module: mod, Version: ver})
	}

	allMods, err := u.proxy.ResolveDeps(mods, u.newModules)
	if err != nil {
		return err
	}

	for _, mod := range allMods {
		if rule, ok := existingRules[mod.Module]; ok {
			rule.SetAttr("version", NewStringExpr(mod.Version))
			continue
		}

		rule := build.NewRule(&build.CallExpr{
			X:    &build.Ident{Name: "go_repo"},
			List: []build.Expr{},
		})
		rule.SetAttr("module", NewStringExpr(mod.Module))
		rule.SetAttr("version", NewStringExpr(mod.Version))
		file.Stmt = append(file.Stmt, rule.Call)
	}
	return saveAndFormatBuildFile(file)
}

// updateDeps updates the dependencies of a build rule based on the imports of its sources
func (u *Update) updateDeps(rule *Rule, rules []*Rule, sources map[string]*GoFile) error {
	srcs := append(rule.srcs, rule.cgoSrcs...)

	done := map[string]struct{}{}
	rule.deps = nil
	for _, src := range srcs {
		f := sources[src]
		for _, i := range f.Imports {
			if knownimports.IsInGoRoot(i) {
				continue
			}
			if _, ok := done[i]; ok {
				continue
			}
			done[i] = struct{}{}

			// TODO update visibility here? Maybe that should be done manually though?
			t := depTarget(u.modules, u.importPath, i, u.thirdPartyDir)
			if t != "" {
				rule.deps = append(rule.deps, t)
				continue
			}

			// Generate a new dep
			mod, err := u.proxy.ResolveModuleForPackage(i)
			if err != nil {
				fmt.Println("error resolving dep for", rule.name, err.Error(), u.importPath)
				continue
			}
			u.newModules = append(u.newModules, mod)
			u.modules = append(u.modules, mod.Module)

			t = depTarget(u.modules, u.importPath, i, u.thirdPartyDir)
			if t != "" {
				rule.deps = append(rule.deps, t)
				continue
			}
		}
	}

	// Add any libraries for the same package as us
	if rule.test {
		pkgName, err := rulePkg(sources, rule)
		if err != nil {
			return err
		}

		for _, libRule := range rules {
			if libRule.test {
				continue
			}
			libPkgName, err := rulePkg(sources, libRule)
			if err != nil {
				return err
			}

			if libPkgName == pkgName {
				rule.deps = append(rule.deps, fmt.Sprintf(":%v", libRule.name))
			}
		}
	}

	return nil
}

// readRulesFromFile reads the existing build rules from the BUILD file
func readRulesFromFile(file *build.File) ([]*Rule, map[string]*build.CallExpr) {
	var rules []*Rule
	calls := map[string]*build.CallExpr{}

	for _, expr := range file.Stmt {
		call, ok := expr.(*build.CallExpr)
		if !ok {
			continue
		}

		rule := callToRule(call)
		rules = append(rules, rule)
		calls[rule.name] = call
	}

	return rules, calls
}

// updateCalls updates the call expressions from the BUILD file
func (u *Update) updateCalls(calls map[string]*build.CallExpr, rules []*Rule, sources map[string]*GoFile) error {
	for _, rule := range rules {
		if err := u.updateDeps(rule, rules, sources); err != nil {
			return err
		}
		call := calls[rule.name]
		populateRule(build.NewRule(call), rule)
	}
	return nil
}

// allocateSources allocates sources to rules. If there's no existing rule, a new rule will be created and returned
// from this function
func allocateSources(pkgDir string, files map[string]*GoFile, unallocated []string, rules []*Rule) ([]*Rule, error) {
	var newRules []*Rule
	for _, src := range unallocated {
		importedFile := files[src]
		var rule *Rule
		for _, r := range append(rules, newRules...) {
			rulePkgName, err := rulePkg(files, r)
			if err != nil {
				return nil, fmt.Errorf("failed to determine package name for //%v:%v: %w", pkgDir, r.name, err)
			}

			if rulePkgName == importedFile.Name && r.test == importedFile.Test {
				rule = r
				break
			}
		}
		if rule == nil {
			name := filepath.Base(pkgDir)
			kind := "go_library"
			if importedFile.Test {
				name = name + "_test"
				kind = "go_test"
			}
			if importedFile.Cmd {
				kind = "go_binary"
				name = "main"
			}
			rule = &Rule{
				name: name,
				kind: kind,
			}
			newRules = append(newRules, rule)
		}

		if importedFile.Cgo {
			rule.cgoSrcs = append(rule.cgoSrcs, src)
		} else {
			rule.srcs = append(rule.srcs, src)
		}
	}
	return newRules, nil
}

// rulePkg checks the first source it finds for a rule and returns the name from the "package name" directive at the top
// of the file
func rulePkg(srcs map[string]*GoFile, rule *Rule) (string, error) {
	var src string

	if len(rule.srcs) > 0 {
		src = rule.srcs[0]
	} else if len(rule.cgoSrcs) > 0 {
		src = rule.cgoSrcs[0]
	} else {
		return "", errors.New("no source files found")
	}

	return srcs[src].Name, nil
}

// unallocatedSources returns all the sources that don't already belong to a rule
func unallocatedSources(srcs map[string]*GoFile, rules []*Rule) []string {
	var ret []string
	for src := range srcs {
		found := false
		for _, rule := range rules {
			if found {
				break
			}
			for _, s := range rule.srcs {
				if s == src {
					found = true
					break
				}
			}
		}
		if !found {
			ret = append(ret, src)
		}
	}
	return ret
}
