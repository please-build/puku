package generate

import (
	"fmt"
	"path/filepath"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"
	"github.com/peterebden/go-cli-init/v5/logging"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/kinds"
	"github.com/please-build/puku/please"
	"github.com/please-build/puku/proxy"
	"github.com/please-build/puku/trie"
)

var log = logging.MustGetLogger()

type Proxy interface {
	ResolveModuleForPackage(pattern string) (*proxy.Module, error)
	ResolveDeps(mods, newMods []*proxy.Module) ([]*proxy.Module, error)
}

type Update struct {
	conf                 *please.Config
	write, usingGoModule bool

	newModules   []*proxy.Module
	modules      []string
	knownImports map[string]string
	installs     *trie.Trie

	paths []string

	proxy Proxy
}

func NewUpdate(write bool, conf *please.Config) *Update {
	return &Update{
		proxy:        proxy.New("https://proxy.golang.org"),
		installs:     trie.New(),
		write:        write,
		knownImports: map[string]string{},
		conf:         conf,
	}
}

// Update updates an existing Please project. It may create new BUILD files, however it tries to respect existing build
// rules, updating them as appropriate.
func (u *Update) Update(paths ...string) error {
	conf, err := config.ReadConfig(".")
	if err != nil {
		return err
	}
	u.paths = paths

	if err := u.readModules(conf); err != nil {
		return fmt.Errorf("failed to read third party rules: %v", err)
	}

	return u.update(conf)
}

// getModules returns the defined third party modules in this project
func (u *Update) readModules(conf *config.Config) error {
	// TODO we probably want to support multiple third party dirs mostly just for our setup in core3
	f, err := parseBuildFile(conf.GetThirdPartyDir(), u.conf.BuildFileNames())
	if err != nil {
		return err
	}

	addInstalls := func(targetName, modName string, installs []string) {
		for _, install := range installs {
			path := filepath.Join(modName, install)
			target := buildTarget(targetName, conf.GetThirdPartyDir(), "")
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
func (u *Update) update(conf *config.Config) error {
	for _, path := range u.paths {
		conf, err := config.ReadConfig(path)
		if err != nil {
			return err
		}

		if conf.Stop {
			return nil
		}

		if err := u.updateOne(conf, path); err != nil {
			return fmt.Errorf("failed to update %v: %v", path, err)
		}
	}

	// Save any new modules we needed back to the third party file
	return u.addNewModules(conf)
}

func (u *Update) updateOne(conf *config.Config, path string) error {
	// Find all the files in the dir
	sources, err := ImportDir(path)
	if err != nil {
		return err
	}

	if len(sources) == 0 {
		return nil
	}

	// Parse the build file
	file, err := parseBuildFile(path, u.conf.BuildFileNames())
	if err != nil {
		return err
	}

	if !u.conf.GoIsPreloaded() {
		ensureSubinclude(file)
	}

	// Read existing rules from file
	rules, calls := u.readRulesFromFile(conf, file, path)

	// Allocate the sources to the rules, creating new rules as necessary
	newRules, srcsWereModded, err := u.allocateSources(path, sources, rules)
	if err != nil {
		return err
	}

	rules = append(rules, newRules...)

	// Update the existing call expressions in the build file
	depsWereModded, err := u.updateDeps(conf, file, calls, rules, sources)
	if err != nil {
		return err
	}

	// If we modified anything, we should format the file
	if srcsWereModded || depsWereModded {
		// Save the file back
		return saveAndFormatBuildFile(file, u.write)
	}
	return nil
}

func (u *Update) addNewModules(conf *config.Config) error {
	file, err := parseBuildFile(conf.GetThirdPartyDir(), u.conf.BuildFileNames())
	if err != nil {
		return err
	}

	// This is a workaround for a bug in Please. It seems we queue up the third party build file for subinclude in order
	// to get at the go_repo rules. This means the go rules aren't preloaded. I don't think this should be the case.
	//
	// TODO figure out why Please needs this subinclude and check for u.goIsPreloaded before calling this once that's
	// 	fixed
	ensureSubinclude(file)

	modified := false

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
		modified = true
	}
	if modified {
		return saveAndFormatBuildFile(file, u.write)
	}
	return nil
}

// updateRuleDeps updates the dependencies of a build rule based on the imports of its sources
func (u *Update) updateRuleDeps(conf *config.Config, rule *rule, rules []*rule, sources map[string]*GoFile) (bool, error) {
	done := map[string]struct{}{}

	// If the rule operates on non-go sources (e.g. .proto sources for proto_library) then we should skip updating
	// its deps.
	if rule.kind.NonGoSources {
		return false, nil
	}

	srcs, err := rule.allSources()
	if err != nil {
		return false, err
	}

	depsBefore := rule.AttrStrings("deps")
	deps := make(map[string]struct{}, len(depsBefore))

	for _, dep := range depsBefore {
		deps[dep] = struct{}{}
	}

	modified := false
	for _, src := range srcs {
		f := sources[src]
		if f == nil {
			rule.removeSrc(src) // The src doesn't exist so remove it from the list of srcs
			continue
		}
		for _, i := range f.Imports {
			if _, ok := done[i]; ok {
				continue
			}
			done[i] = struct{}{}

			// If the dep is provided by the kind (i.e. the build def adds it) then skip this import

			dep, err := u.resolveImport(conf, i)
			if err != nil {
				log.Warningf("couldn't resolve %q for %v: %v", i, rule.label(), err)
				continue
			}
			if dep == "" {
				continue
			}
			if rule.kind.IsProvided(dep) {
				continue
			}

			dep = shorten(rule.dir, dep)

			if _, ok := deps[dep]; !ok {
				modified = true
				deps[dep] = struct{}{}
			}
		}
	}

	// Add any libraries for the same package as us
	if rule.kind.Type == kinds.Test && !rule.isExternal() {
		pkgName, err := rulePkg(sources, rule)
		if err != nil {
			return false, err
		}

		for _, libRule := range rules {
			if libRule.kind.Type == kinds.Test {
				continue
			}
			libPkgName, err := rulePkg(sources, libRule)
			if err != nil {
				return false, err
			}

			if libPkgName != pkgName {
				continue
			}

			t := libRule.localLabel()
			if _, ok := deps[t]; !ok {
				modified = true
				deps[t] = struct{}{}
			}
		}
	}

	depSlice := make([]string, 0, len(deps))
	for dep := range deps {
		depSlice = append(depSlice, dep)
	}

	rule.setOrDeleteAttr("deps", depSlice)

	return modified, nil
}

// shorten will shorten lables to the local package
func shorten(pkg, label string) string {
	if strings.HasPrefix(label, "///") || strings.HasPrefix(label, "@") {
		return label
	}

	return labels.Shorten(label, pkg)
}

// readRulesFromFile reads the existing build rules from the BUILD file
func (u *Update) readRulesFromFile(conf *config.Config, file *build.File, pkgDir string) ([]*rule, map[string]*build.Rule) {
	var rules []*rule
	calls := map[string]*build.Rule{}

	for _, expr := range file.Rules("") {
		kind := conf.GetKind(expr.Kind())
		if kind == nil {
			continue
		}
		rule := newRule(expr, kind, pkgDir)
		rules = append(rules, rule)
		calls[rule.Name()] = expr
	}

	return rules, calls
}

// updateDeps updates the existing rules and creates any new rules in the BUILD file
func (u *Update) updateDeps(conf *config.Config, file *build.File, ruleExprs map[string]*build.Rule, rules []*rule, sources map[string]*GoFile) (bool, error) {
	modified := false
	for _, rule := range rules {
		if _, ok := ruleExprs[rule.Name()]; !ok {
			file.Stmt = append(file.Stmt, rule.Call)
		}
		modded, err := u.updateRuleDeps(conf, rule, rules, sources)
		if err != nil {
			return false, err
		}
		modified = modded || modified
	}
	return modified, nil
}

// allocateSources allocates sources to rules. If there's no existing rule, a new rule will be created and returned
// from this function
func (u *Update) allocateSources(pkgDir string, sources map[string]*GoFile, rules []*rule) ([]*rule, bool, error) {
	unallocated, err := unallocatedSources(sources, rules)
	if err != nil {
		return nil, false, err
	}

	var newRules []*rule
	for _, src := range unallocated {
		importedFile := sources[src]
		if importedFile == nil {
			continue // Something went wrong and we haven't imported the file don't try to allocate it
		}
		var rule *rule
		for _, r := range append(rules, newRules...) {
			if r.kind.Type != importedFile.kindType() {
				continue
			}

			rulePkgName, err := rulePkg(sources, r)
			if err != nil {
				return nil, false, fmt.Errorf("failed to determine package name for //%v:%v: %w", pkgDir, r.Name(), err)
			}

			// Find a rule that's for thhe same package and of the same kind (i.e. bin, lib, test)
			// NB: we return when we find the first one so if there are multiple options, we will pick one essentially at
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
			rule = newRule(newRuleExpr(kind, name), kinds.DefaultKinds[kind], pkgDir)
			if importedFile.IsExternal(filepath.Join(u.conf.ImportPath(), pkgDir)) {
				rule.setExternal()
			}
			newRules = append(newRules, rule)
		}

		rule.addSrc(src)
	}
	return newRules, len(unallocated) > 0, nil
}

// rulePkg checks the first source it finds for a rule and returns the name from the "package name" directive at the top
// of the file
func rulePkg(srcs map[string]*GoFile, rule *rule) (string, error) {
	// This is a safe bet if we can't use the source files to figure this out.
	if rule.kind.NonGoSources {
		return rule.Name(), nil
	}

	s, err := rule.allSources()
	if err != nil {
		return "", err
	}
	if len(s) <= 0 { // there is a rule with no sources yet we can't determine the package
		return "", nil
	}

	return srcs[s[0]].Name, nil
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
