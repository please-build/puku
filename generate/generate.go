package generate

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/proxy"
	"github.com/please-build/puku/trie"

	"github.com/bazelbuild/buildtools/build"
	"github.com/peterebden/go-cli-init/v5/logging"
)

var log = logging.MustGetLogger()

type KindType int

const (
	KindTypeLib KindType = iota
	KindTypeTest
	KindTypeBin
)

var defaultKinds = map[string]KindType{
	"go_library": KindTypeLib,
	"go_binary":  KindTypeBin,
	"go_test":    KindTypeTest,
}

type Update struct {
	plzPath, thirdPartyDir string
	buildFileNames         []string
	kinds                  map[string]KindType

	importPath                   string
	newModules                   []*proxy.Module
	modules                      []string
	knownImports                 map[string]string
	installs                     *trie.Trie
	usingGoModule, goIsPreloaded bool

	paths []string

	proxy *proxy.Proxy
}

func NewUpdate(plzPath, thirdPartyDir string) *Update {
	return &Update{
		plzPath:       plzPath,
		thirdPartyDir: thirdPartyDir,
		proxy:         proxy.New("https://proxy.golang.org"),
		kinds:         defaultKinds,
		installs:      trie.New(),
		knownImports:  map[string]string{},
	}
}

// Update updates an existing Please project. It may create new BUILD files, however it tries to respect existing build
// rules, updating them as appropriate.
func (u *Update) Update(paths []string) error {
	u.paths = paths

	var err error

	c, err := config.QueryConfig(u.plzPath)
	if err != nil {
		return fmt.Errorf("failed to query config: %w", err)
	}

	u.importPath = c.ImportPath()
	u.buildFileNames = c.BuildFileNames()
	u.goIsPreloaded = c.GoIsPreloaded()

	if err := u.readModules(); err != nil {
		return fmt.Errorf("failed to read third party rules: %v", err)
	}

	return u.update()
}

// getModules returns the defined third party modules in this project
func (u *Update) readModules() error {
	// TODO we probably want to support multiple third party dirs mostly just for our setup in core3
	f, err := parseBuildFile(u.thirdPartyDir, u.buildFileNames)
	if err != nil {
		return err
	}

	addInstalls := func(targetName, modName string, installs []string) {
		for _, install := range installs {
			path := filepath.Join(modName, install)
			target := buildTarget(targetName, u.thirdPartyDir, "")
			u.installs.Add(path, target)
		}
	}

	for _, repoRule := range f.Rules("go_repo") {
		module := repoRule.AttrString("module")
		u.modules = append(u.modules, module)

		installs := repoRule.AttrStrings("install")
		if len(installs) > 0 {
			addInstalls(repoRule.Name(), module, installs)
		}
	}

	goMods := f.Rules("go_module")
	u.usingGoModule = len(goMods) > 0

	for _, mod := range goMods {
		module := mod.AttrString("module")
		installs := mod.AttrStrings("install")
		addInstalls(mod.Name(), module, installs)
	}

	return nil
}

// update loops through the provided paths, updating and creating any build rules it finds.
func (u *Update) update() error {
	for _, path := range u.paths {
		if strings.HasSuffix(path, "...") {
			p := filepath.Clean(strings.TrimSuffix(path, "..."))
			if err := u.updateAll(p); err != nil {
				return fmt.Errorf("failed to update %v: %v", path, err)
			}
		} else if err := u.updateOne(path); err != nil {
			return fmt.Errorf("failed to update %v: %v", path, err)
		}
	}

	// Save any new modules we needed back to the third party file
	return u.addNewModules()
}

func (u *Update) updateAll(path string) error {
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			if d.Name() == "plz-out" {
				return filepath.SkipDir
			}
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			if err := u.updateOne(path); err != nil {
				return err
			}
		}
		return nil
	})
}

func (u *Update) updateOne(path string) error {
	// Find all the files in the dir
	sources, err := ImportDir(path)
	if err != nil {
		return err
	}

	if len(sources) == 0 {
		return nil
	}

	// Parse the build file
	file, err := parseBuildFile(path, u.buildFileNames)
	if err != nil {
		return err
	}

	if !u.goIsPreloaded {
		ensureSubinclude(file)
	}

	// Read existing rules from file
	rules, calls := u.readRulesFromFile(file, path)

	// Allocate the sources to the rules, creating new rules as necessary
	newRules, err := u.allocateSources(path, sources, rules)
	if err != nil {
		return err
	}

	rules = append(rules, newRules...)

	// Update the existing call expressions in the build file
	if err := u.updateFile(file, calls, rules, sources); err != nil {
		return err
	}

	// Save the file back
	return saveAndFormatBuildFile(file)
}

func (u *Update) addNewModules() error {
	file, err := parseBuildFile(u.thirdPartyDir, u.buildFileNames)
	if err != nil {
		return err
	}

	ensureSubinclude(file)

	var mods []*proxy.Module
	existingRules := make(map[string]*build.Rule)
	for _, rule := range file.Rules("go_repo") {
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
			// Modules might be using go_mod_download, which we don't handle.
			if rule.Attr("version") != nil {
				rule.SetAttr("version", newStringExpr(mod.Version))
			}
			continue
		}

		rule := build.NewRule(&build.CallExpr{
			X:    &build.Ident{Name: "go_repo"},
			List: []build.Expr{},
		})
		rule.SetAttr("module", newStringExpr(mod.Module))
		rule.SetAttr("version", newStringExpr(mod.Version))
		file.Stmt = append(file.Stmt, rule.Call)
	}
	return saveAndFormatBuildFile(file)
}

// updateDeps updates the dependencies of a build rule based on the imports of its sources
func (u *Update) updateDeps(rule *rule, rules []*rule, sources map[string]*GoFile) error {
	done := map[string]struct{}{}
	srcs, err := rule.allSources()
	if err != nil {
		return err
	}

	var deps []string
	for _, src := range srcs {
		f := sources[src]
		for _, i := range f.Imports {
			if _, ok := done[i]; ok {
				continue
			}
			done[i] = struct{}{}

			dep, err := u.resolveImport(i)
			if err != nil {
				log.Warningf("couldn't resolve %q for %v: %v", i, rule.label(), err)
				continue
			}
			if dep != "" {
				deps = append(deps, dep)
			}
		}
	}

	// Add any libraries for the same package as us
	if rule.kindType == KindTypeTest {
		pkgName, err := rulePkg(sources, rule)
		if err != nil {
			return err
		}

		for _, libRule := range rules {
			if libRule.kindType == KindTypeTest {
				continue
			}
			libPkgName, err := rulePkg(sources, libRule)
			if err != nil {
				return err
			}

			if libPkgName == pkgName {
				deps = append(deps, libRule.localLabel())
			}
		}
	}

	rule.setOrDeleteAttr("deps", deps)

	return nil
}

// readRulesFromFile reads the existing build rules from the BUILD file
func (u *Update) readRulesFromFile(file *build.File, pkgDir string) ([]*rule, map[string]*build.Rule) {
	var rules []*rule
	calls := map[string]*build.Rule{}

	for _, expr := range file.Rules("") {
		kindType, ok := u.kinds[expr.Kind()]
		if !ok {
			continue
		}
		rule := newRule(expr, kindType, pkgDir)
		rules = append(rules, rule)
		calls[rule.Name()] = expr
	}

	return rules, calls
}

// updateFile updates the existing rules and creates any new rules in the BUILD file
func (u *Update) updateFile(file *build.File, ruleExprs map[string]*build.Rule, rules []*rule, sources map[string]*GoFile) error {
	for _, rule := range rules {
		if _, ok := ruleExprs[rule.Name()]; !ok {
			file.Stmt = append(file.Stmt, rule.Call)
		}
		if err := u.updateDeps(rule, rules, sources); err != nil {
			return err
		}
	}
	return nil
}

// allocateSources allocates sources to rules. If there's no existing rule, a new rule will be created and returned
// from this function
func (u *Update) allocateSources(pkgDir string, sources map[string]*GoFile, rules []*rule) ([]*rule, error) {
	unallocated, err := unallocatedSources(sources, rules)
	if err != nil {
		return nil, err
	}

	var newRules []*rule
	for _, src := range unallocated {
		importedFile := sources[src]
		var rule *rule
		for _, r := range append(rules, newRules...) {
			if r.kindType != importedFile.kindType() {
				continue
			}

			rulePkgName, err := rulePkg(sources, r)
			if err != nil {
				return nil, fmt.Errorf("failed to determine package name for //%v:%v: %w", pkgDir, r.Name(), err)
			}

			// Find a rule that's for thhe same package and of the same kind (i.e. bin, lib, test)
			// NB: we return when we find the first one so if there are multiple options, we will pick on essentially at
			//     random.
			if rulePkgName == "" || rulePkgName == importedFile.Name {
				rule = r
				break
			}
		}
		if rule == nil {
			name := filepath.Base(pkgDir)
			kind := "go_library"
			if importedFile.IsTest() {
				name = name + "_test"
				kind = "go_test"
			}
			if importedFile.IsCmd() {
				kind = "go_binary"
				name = "main"
			}
			rule = newRule(newRuleExpr(kind, name), importedFile.kindType(), pkgDir)
			if importedFile.IsExternal(filepath.Join(u.importPath, pkgDir)) {
				rule.setExternal()
			}
			newRules = append(newRules, rule)
		}

		rule.addSrc(src)

	}
	return newRules, nil
}

// rulePkg checks the first source it finds for a rule and returns the name from the "package name" directive at the top
// of the file
func rulePkg(srcs map[string]*GoFile, rule *rule) (string, error) {
	var src string

	s, err := rule.allSources()
	if err != nil {
		return "", err
	}
	if len(s) > 0 {
		src = s[0]
	} else {
		return "", nil
	}

	return srcs[src].Name, nil
}

// unallocatedSources returns all the sources that don't already belong to a rule
func unallocatedSources(srcs map[string]*GoFile, rules []*rule) ([]string, error) {
	var ret []string
	for src := range srcs {
		found := false
		for _, rule := range rules {
			if found {
				break
			}

			ruleSrcs, err := rule.allSources()
			if err != nil {
				return nil, err
			}
			for _, s := range ruleSrcs {
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
	return ret, nil
}
