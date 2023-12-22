package generate

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/please-build/buildtools/build"
	"github.com/please-build/buildtools/labels"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/eval"
	"github.com/please-build/puku/glob"
	"github.com/please-build/puku/graph"
	"github.com/please-build/puku/kinds"
	"github.com/please-build/puku/licences"
	"github.com/please-build/puku/logging"
	"github.com/please-build/puku/please"
	"github.com/please-build/puku/proxy"
	"github.com/please-build/puku/sync"
	"github.com/please-build/puku/trie"
)

var log = logging.GetLogger()

type Proxy interface {
	ResolveModuleForPackage(pattern string) (*proxy.Module, error)
	ResolveDeps(mods, newMods []*proxy.Module) ([]*proxy.Module, error)
}

type Update struct {
	plzConf              *please.Config
	write, usingGoModule bool

	graph *graph.Graph

	newModules      []*proxy.Module
	modules         []string
	resolvedImports map[string]string
	installs        *trie.Trie
	eval            *eval.Eval

	paths []string

	proxy    Proxy
	licences *licences.Licenses
	sync     *sync.Sync
}

func NewUpdate(write bool, plzConf *please.Config) *Update {
	return NewUpdateWithGraph(write, plzConf, graph.New(plzConf.BuildFileNames()))
}

// NewUpdateWithGraph is like NewUpdate but lets us inject a graph which is useful to do testing.
func NewUpdateWithGraph(write bool, conf *please.Config, g *graph.Graph) *Update {
	p := proxy.New(proxy.DefaultURL)
	l := licences.New(p, g)
	return &Update{
		proxy:           p,
		licences:        l,
		installs:        trie.New(),
		write:           write,
		resolvedImports: map[string]string{},
		plzConf:         conf,
		eval:            eval.New(glob.New()),
		graph:           g,
		sync:            sync.New(conf, g, l, write),
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

	if err := u.readAllModules(conf); err != nil {
		return fmt.Errorf("failed to read third party rules: %v", err)
	}

	if err := u.update(conf); err != nil {
		return err
	}

	if err := u.sync.Sync(); err != nil {
		return fmt.Errorf("failed to sync go.mod: %w", err)
	}

	return u.graph.FormatFiles(u.write, os.Stdout)
}

func (u *Update) readAllModules(conf *config.Config) error {
	return filepath.WalkDir(conf.GetThirdPartyDir(), func(path string, info fs.DirEntry, err error) error {
		for _, buildFileName := range u.plzConf.BuildFileNames() {
			if info.Name() == buildFileName {
				file, err := u.graph.LoadFile(filepath.Dir(path))
				if err != nil {
					return err
				}

				if err := u.readModules(file); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// readModules returns the defined third party modules in this project
func (u *Update) readModules(file *build.File) error {
	addInstalls := func(targetName, modName string, installs []string) {
		for _, install := range installs {
			path := filepath.Join(modName, install)
			target := BuildTarget(targetName, file.Pkg, "")
			u.installs.Add(path, target)
		}
	}

	for _, repoRule := range file.Rules("go_repo") {
		module := repoRule.AttrString("module")
		u.modules = append(u.modules, module)

		installs := repoRule.AttrStrings("install")
		if len(installs) > 0 {
			addInstalls(repoRule.Name(), module, installs)
		}
	}

	goMods := file.Rules("go_module")
	u.usingGoModule = len(goMods) > 0 || u.usingGoModule

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

	// Parse the build file
	file, err := u.graph.LoadFile(path)
	if err != nil {
		return err
	}

	if !u.plzConf.GoIsPreloaded() && conf.ShouldEnsureSubincludes() {
		edit.EnsureSubinclude(file)
	}

	// Read existing rules from file
	rules, calls := u.readRulesFromFile(conf, file, path)

	// Allocate the sources to the rules, creating new rules as necessary
	newRules, err := u.allocateSources(conf, path, sources, rules)
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

	if !u.plzConf.GoIsPreloaded() && conf.ShouldEnsureSubincludes() {
		edit.EnsureSubinclude(file)
	}

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
		ls, err := u.licences.Get(mod.Module, mod.Version)
		if err != nil {
			return fmt.Errorf("failed to get license for mod %v: %v", mod.Module, err)
		}
		file.Stmt = append(file.Stmt, edit.NewGoRepoRule(mod.Module, mod.Version, "", ls))
	}
	return nil
}

// allSources calculates the sources for a target. It will evaluate the source list resolving globs, and building any
// srcs that are other build targets.
//
// passedSources is a slice of filepaths, which contains source files passed to the rule, after resolving globs and
// building any targets. These source files can be looked up in goFiles, if they exist.
//
// goFiles contains a mapping of source files to their GoFile. This map might be missing entries from passedSources, if
// the source doesn't actually exist. In which case, this should be removed from the rule, as the user likely deleted
// the file.
func (u *Update) allSources(conf *config.Config, r *rule, sourceMap map[string]*GoFile) (passedSources []string, goFiles map[string]*GoFile, err error) {
	srcs, err := u.eval.BuildSources(conf.GetPlzPath(), r.dir, r.Rule)
	if err != nil {
		return nil, nil, err
	}

	sources := make(map[string]*GoFile, len(srcs))
	for _, src := range srcs {
		if file, ok := sourceMap[src]; ok {
			sources[src] = file
			continue
		}

		// These are generated sources in plz-out/gen
		f, err := importFile(".", src)
		if err != nil {
			continue
		}
		sources[src] = f
	}
	return srcs, sources, nil
}

// updateRuleDeps updates the dependencies of a build rule based on the imports of its sources
func (u *Update) updateRuleDeps(conf *config.Config, rule *rule, rules []*rule, packageFiles map[string]*GoFile) error {
	done := map[string]struct{}{}

	// If the rule operates on non-go source files (e.g. *.proto for proto_library) then we should skip updating
	// it as we can't determine its deps from sources this way.
	if rule.kind.NonGoSources {
		return nil
	}

	srcs, targetFiles, err := u.allSources(conf, rule, packageFiles)
	if err != nil {
		return err
	}

	label := BuildTarget(rule.Name(), rule.dir, "")

	deps := map[string]struct{}{}
	for _, src := range srcs {
		f := targetFiles[src]
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
		pkgName, err := u.rulePkg(conf, packageFiles, rule)
		if err != nil {
			return err
		}

		for _, libRule := range rules {
			if libRule.kind.Type == kinds.Test {
				continue
			}
			libPkgName, err := u.rulePkg(conf, packageFiles, libRule)
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
func (u *Update) allocateSources(conf *config.Config, pkgDir string, sources map[string]*GoFile, rules []*rule) ([]*rule, error) {
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

			rulePkgName, err := u.rulePkg(conf, sources, r)
			if err != nil {
				return nil, fmt.Errorf("failed to determine package name for //%v:%v: %w", pkgDir, r.Name(), err)
			}

			// Find a rule that's for the same package and of the same kind (i.e. bin, lib, test)
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
			if importedFile.IsExternal(filepath.Join(u.plzConf.ImportPath(), pkgDir)) {
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
func (u *Update) rulePkg(conf *config.Config, srcs map[string]*GoFile, rule *rule) (string, error) {
	// This is a safe bet if we can't use the source files to figure this out.
	if rule.kind.NonGoSources {
		return rule.Name(), nil
	}

	ss, srcs, err := u.allSources(conf, rule, srcs)
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

			ruleSrcs, err := u.eval.EvalGlobs(rule.dir, rule.Rule)
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
