package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"
	"github.com/peterebden/go-cli-init/v5/logging"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/glob"
	"github.com/please-build/puku/graph"
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

	graph *graph.Graph

	newModules   []*proxy.Module
	modules      []string
	knownImports map[string]string
	installs     *trie.Trie

	globber *glob.Globber

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
		globber:      glob.New(),
		graph:        graph.New(conf.BuildFileNames()),
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

	if err := u.update(conf); err != nil {
		return err
	}
	return u.graph.FormatFiles(u.write, os.Stdout)
}

// getModules returns the defined third party modules in this project
func (u *Update) readModules(conf *config.Config) error {
	// TODO we probably want to support multiple third party dirs mostly just for our setup in core3
	f, err := u.graph.LoadFile(conf.GetThirdPartyDir())
	if err != nil {
		return err
	}

	addInstalls := func(targetName, modName string, installs []string) {
		for _, install := range installs {
			path := filepath.Join(modName, install)
			target := BuildTarget(targetName, conf.GetThirdPartyDir(), "")
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
		if len(installs) == 0 {
			installs = []string{"."}
		}
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
	file, err := u.graph.LoadFile(path)
	if err != nil {
		return err
	}

	if !u.conf.GoIsPreloaded() {
		edit.EnsureSubinclude(file)
	}

	// Read existing rules from file
	rules, calls := u.readRulesFromFile(conf, file, path)

	// Allocate the sources to the rules, creating new rules as necessary
	newRules, err := u.allocateSources(path, sources, rules)
	if err != nil {
		return err
	}

	rules = append(rules, newRules...)

	// Update the existing call expressions in the build file
	return u.updateDeps(conf, file, calls, rules, sources)
}

func (u *Update) addNewModules(conf *config.Config) error {
	file, err := u.graph.LoadFile(conf.GetThirdPartyDir())
	if err != nil {
		return err
	}

	// This is a workaround for a bug in Please. It seems we queue up the third party build file for subinclude in order
	// to get at the go_repo rules. This means the go rules aren't preloaded. I don't think this should be the case.
	//
	// TODO figure out why Please needs this subinclude and check for u.goIsPreloaded before calling this once that's
	// 	fixed
	edit.EnsureSubinclude(file)

	goRepos := file.Rules("go_repo")
	mods := make([]*proxy.Module, 0, len(goRepos))
	existingRules := make(map[string]*build.Rule)
	for _, rule := range goRepos {
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
				rule.SetAttr("version", edit.NewStringExpr(mod.Version))
			}
			continue
		}

		rule := build.NewRule(&build.CallExpr{
			X:    &build.Ident{Name: "go_repo"},
			List: []build.Expr{},
		})
		rule.SetAttr("module", edit.NewStringExpr(mod.Module))
		rule.SetAttr("version", edit.NewStringExpr(mod.Version))
		file.Stmt = append(file.Stmt, rule.Call)
	}
	return nil
}

func (u *Update) allSources(r *rule) ([]string, error) {
	args := r.parseGlob()
	if args == nil {
		return r.AttrStrings("srcs"), nil
	}

	return u.globber.Glob(r.dir, args)
}

// updateRuleDeps updates the dependencies of a build rule based on the imports of its sources
func (u *Update) updateRuleDeps(conf *config.Config, rule *rule, rules []*rule, sources map[string]*GoFile) error {
	done := map[string]struct{}{}

	// If the rule operates on non-go sources (e.g. .proto sources for proto_library) then we should skip updating
	// its deps.
	if rule.kind.NonGoSources {
		return nil
	}

	srcs, err := u.allSources(rule)
	if err != nil {
		return err
	}

	label := BuildTarget(rule.Name(), rule.dir, "")

	deps := map[string]struct{}{}
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
				deps[dep] = struct{}{}
			}
		}
	}

	// Add any libraries for the same package as us
	if rule.kind.Type == kinds.Test && !rule.isExternal() {
		pkgName, err := u.rulePkg(sources, rule)
		if err != nil {
			return err
		}

		for _, libRule := range rules {
			if libRule.kind.Type == kinds.Test {
				continue
			}
			libPkgName, err := u.rulePkg(sources, libRule)
			if err != nil {
				return err
			}

			if libPkgName != pkgName {
				continue
			}

			t := libRule.localLabel()
			if _, ok := deps[t]; !ok {
				deps[t] = struct{}{}
			}
		}
	}

	depSlice := make([]string, 0, len(deps))
	for dep := range deps {
		u.graph.EnsureVisibility(label, dep)
		depSlice = append(depSlice, dep)
	}

	rule.setOrDeleteAttr("deps", depSlice)

	return nil
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
	ruleExprs := file.Rules("")
	rules := make([]*rule, 0, len(ruleExprs))
	calls := map[string]*build.Rule{}

	for _, expr := range ruleExprs {
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
func (u *Update) updateDeps(conf *config.Config, file *build.File, ruleExprs map[string]*build.Rule, rules []*rule, sources map[string]*GoFile) error {
	for _, rule := range rules {
		if _, ok := ruleExprs[rule.Name()]; !ok {
			file.Stmt = append(file.Stmt, rule.Call)
		}
		if err := u.updateRuleDeps(conf, rule, rules, sources); err != nil {
			return err
		}
	}
	return nil
}

// allocateSources allocates sources to rules. If there's no existing rule, a new rule will be created and returned
// from this function
func (u *Update) allocateSources(pkgDir string, sources map[string]*GoFile, rules []*rule) ([]*rule, error) {
	unallocated, err := u.unallocatedSources(sources, rules)
	if err != nil {
		return nil, err
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

			rulePkgName, err := u.rulePkg(sources, r)
			if err != nil {
				return nil, fmt.Errorf("failed to determine package name for //%v:%v: %w", pkgDir, r.Name(), err)
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
				name += "_test"
				kind = "go_test"
			}
			if importedFile.IsCmd() {
				kind = "go_binary"
				name = "main"
			}
			rule = newRule(edit.NewRuleExpr(kind, name), kinds.DefaultKinds[kind], pkgDir)
			if importedFile.IsExternal(filepath.Join(u.conf.ImportPath(), pkgDir)) {
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
func (u *Update) rulePkg(srcs map[string]*GoFile, rule *rule) (string, error) {
	// This is a safe bet if we can't use the source files to figure this out.
	if rule.kind.NonGoSources {
		return rule.Name(), nil
	}

	ss, err := u.allSources(rule)
	if err != nil {
		return "", err
	}

	for _, s := range ss {
		if src, ok := srcs[s]; ok {
			return src.Name, nil
		}
	}

	return "", nil
}

// unallocatedSources returns all the sources that don't already belong to a rule
func (u *Update) unallocatedSources(srcs map[string]*GoFile, rules []*rule) ([]string, error) {
	var ret []string
	for src := range srcs {
		found := false
		for _, rule := range rules {
			if found {
				break
			}

			ruleSrcs, err := u.allSources(rule)
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
