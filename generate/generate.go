package generate

import (
	"fmt"
	"github.com/please-build/puku/config"
	"github.com/please-build/puku/kinds"
	"github.com/please-build/puku/please"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/please-build/puku/proxy"
	"github.com/please-build/puku/trie"

	"github.com/bazelbuild/buildtools/build"
	"github.com/peterebden/go-cli-init/v5/logging"
)

var log = logging.MustGetLogger()

type Proxy interface {
	ResolveModuleForPackage(pattern string) (*proxy.Module, error)
	ResolveDeps(mods, newMods []*proxy.Module) ([]*proxy.Module, error)
}

type Update struct {
	importPath    string
	newModules    []*proxy.Module
	modules       []string
	knownImports  map[string]string
	installs      *trie.Trie
	usingGoModule bool

	paths []string

	proxy          Proxy
	buildFileNames []string
}

func NewUpdate() *Update {
	return &Update{
		proxy:        proxy.New("https://proxy.golang.org"),
		installs:     trie.New(),
		knownImports: map[string]string{},
	}
}

// Update updates an existing Please project. It may create new BUILD files, however it tries to respect existing build
// rules, updating them as appropriate.
func (u *Update) Update(paths []string) error {
	conf, err := config.ReadConfig(".")
	if err != nil {
		return err
	}
	u.paths = paths

	plzConf, err := please.QueryConfig(conf.GetPlzPath())
	if err != nil {
		return fmt.Errorf("failed to query config: %w", err)
	}

	u.importPath = plzConf.ImportPath()
	u.buildFileNames = plzConf.BuildFileNames()

	if err := u.readModules(conf); err != nil {
		return fmt.Errorf("failed to read third party rules: %v", err)
	}

	return u.update(conf)
}

// getModules returns the defined third party modules in this project
func (u *Update) readModules(conf *config.Config) error {
	// TODO we probably want to support multiple third party dirs mostly just for our setup in core3
	f, err := parseBuildFile(conf.GetThirdPartyDir(), u.buildFileNames)
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
		if strings.HasSuffix(path, "...") {
			p := filepath.Clean(strings.TrimSuffix(path, "..."))
			if err := u.updateAll(p); err != nil {
				return fmt.Errorf("failed to update %v: %v", path, err)
			}
		} else if _, err := u.updateOne(path); err != nil {
			return fmt.Errorf("failed to update %v: %v", path, err)
		}
	}

	// Save any new modules we needed back to the third party file
	return u.addNewModules(conf)
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
			if cont, err := u.updateOne(path); err != nil {
				return err
			} else if !cont {
				return filepath.SkipDir
			}
		}
		return nil
	})
}

func (u *Update) updateOne(path string) (cont bool, err error) {
	conf, err := config.ReadConfig(path)
	if err != nil {
		return false, nil
	}

	if conf.Stop {
		return false, nil
	}

	// Find all the files in the dir
	sources, err := ImportDir(path)
	if err != nil {
		return false, err
	}

	if len(sources) == 0 {
		return true, nil
	}

	// Parse the build file
	file, err := parseBuildFile(path, u.buildFileNames)
	if err != nil {
		return false, err
	}

	// Read existing rules from file
	rules, calls := u.readRulesFromFile(conf, file, path)

	// Allocate the sources to the rules, creating new rules as necessary
	newRules, err := u.allocateSources(path, sources, rules)
	if err != nil {
		return false, err
	}

	rules = append(rules, newRules...)

	// Update the existing call expressions in the build file
	if err := u.updateFile(conf, file, calls, rules, sources); err != nil {
		return false, err
	}

	// Save the file back
	return true, saveAndFormatBuildFile(file)
}

func (u *Update) addNewModules(conf *config.Config) error {
	file, err := parseBuildFile(conf.GetThirdPartyDir(), u.buildFileNames)
	if err != nil {
		return err
	}

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
func (u *Update) updateDeps(conf *config.Config, rule *rule, rules []*rule, sources map[string]*GoFile) error {
	done := map[string]struct{}{}

	// If the rule operates on non-go sources (e.g. .proto sources for proto_library) then we should skip updating
	// its deps.
	if rule.kind.NonGoSources {
		return nil
	}

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

			// If the dep is provided by the kind (i.e. the build def adds it) then skip this import

			dep, err := u.resolveImport(conf, i)
			if err != nil {
				log.Warningf("couldn't resolve %q for %v: %v", i, rule.label(), err)
				continue
			}
			if dep != "" {
				if rule.kind.IsProvided(dep) {
					continue
				}
				deps = append(deps, dep)
			}
		}
	}

	// Add any libraries for the same package as us
	if rule.kind.Type == kinds.Test {
		pkgName, err := rulePkg(sources, rule)
		if err != nil {
			return err
		}

		for _, libRule := range rules {
			if libRule.kind.Type == kinds.Test {
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

// updateFile updates the existing rules and creates any new rules in the BUILD file
func (u *Update) updateFile(conf *config.Config, file *build.File, ruleExprs map[string]*build.Rule, rules []*rule, sources map[string]*GoFile) error {
	for _, rule := range rules {
		if _, ok := ruleExprs[rule.Name()]; !ok {
			file.Stmt = append(file.Stmt, rule.Call)
		}
		if err := u.updateDeps(conf, rule, rules, sources); err != nil {
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
			if r.kind.Type != importedFile.kindType() {
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
			rule = newRule(newRuleExpr(kind, name), kinds.DefaultKinds[kind], pkgDir)
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

	// This is a safe bet if we can't use the source files to figure this out.
	if rule.kind.NonGoSources {
		return rule.Name(), nil
	}

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
