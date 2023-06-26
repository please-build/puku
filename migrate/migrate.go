package migrate

import (
	"os"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"
	"golang.org/x/mod/semver"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/generate"
	"github.com/please-build/puku/graph"
	"github.com/please-build/puku/please"
)

func New(conf *config.Config, plzConf *please.Config) *Migrate {
	return &Migrate{
		graph:            graph.New(plzConf.BuildFileNames()),
		thirdPartyFolder: conf.GetThirdPartyDir(),
		mods:             map[string]*module{},
		moduleRules:      map[string]*moduleParts{},
	}
}

type Migrate struct {
	graph            *graph.Graph
	thirdPartyFolder string
	mods             map[string]*module
	moduleRules      map[string]*moduleParts
}

// targetRule represents the rule expr in a pkg
type targetRule struct {
	pkg  string
	rule *build.Rule
}

type moduleParts struct {
	downloads []*targetRule
	goModules []*targetRule
}

type module struct {
	// The name of this rule if there are no aliases
	name string
	// The module name to compile as (passed to go_repo)
	moduleName string
	// If this module should use a download rule rather than version (to handle replacemenets, patches etc.)
	download *build.Rule
	version  string
	install  []string
	patch    []string
	aliases  []labels.Label
}

func (m *Migrate) Migrate(write bool, paths ...string) error {
	for _, path := range paths {
		f, err := m.graph.LoadFile(path)
		if err != nil {
			return err
		}

		// Read all the rules related to each module from all the files into one place. This gives us the ability to
		// make the call on whether we need to keep the download rule e.g. for a patch, or a module replacement
		if err := m.readModuleRules(f, path); err != nil {
			return err
		}

		// Delete all the rules from the files
		f.DelRules("go_module", "")
		f.DelRules("go_mod_download", "")
	}
	m.convertModuleRulesToRepoRules()

	// Now we can generate all the rules we need
	if err := m.genRules(); err != nil {
		return err
	}
	return m.graph.FormatFiles(write, os.Stdout)
}

func (m *Migrate) genRules() error {
	thirdPartyGo, err := m.graph.LoadFile(m.thirdPartyFolder)
	if err != nil {
		return err
	}

	for _, mod := range m.mods {
		// This is the default for go_module which isn't true for go_repo
		if len(mod.install) == 0 {
			mod.install = []string{"."}
		}

		// Remove internal installs as these are no longer required
		installs := make([]string, 0, len(mod.install))
		for _, i := range mod.install {
			if strings.HasPrefix(i, "internal") {
				continue
			}
			installs = append(installs, i)
		}

		for _, alias := range mod.aliases {
			targetFile, err := m.graph.LoadFile(alias.Package)
			if err != nil {
				return err
			}
			targetFile.Stmt = append(targetFile.Stmt, newAlias(alias.Target, mod.moduleName, m.thirdPartyFolder))
		}

		// If we kept the download rule here, we should use that instead of generating one
		if mod.download != nil {
			repoRule := newGoRepoRule(mod.moduleName, "", ":"+mod.download.Name(), mod.name, installs, nil)

			if len(mod.patch) > 0 {
				mod.download.SetAttr("patch", edit.NewStringList(mod.patch))
			}

			thirdPartyGo.Stmt = append(thirdPartyGo.Stmt, mod.download.Call)
			thirdPartyGo.Stmt = append(thirdPartyGo.Stmt, repoRule)
		} else {
			thirdPartyGo.Stmt = append(thirdPartyGo.Stmt, newGoRepoRule(mod.moduleName, mod.version, "", mod.name, installs, mod.patch))
		}
	}
	return nil
}

func (m *Migrate) convertModuleRulesToRepoRules() {
	for modName, rules := range m.moduleRules {
		version := ""
		var patches []string
		for _, rule := range append(rules.goModules, rules.downloads...) {
			v := rule.rule.AttrString("version")
			if v == "" {
				continue
			}
			if version == "" || semver.Compare(version, v) < 0 {
				version = v
			}
			patches = append(patches, rule.rule.AttrStrings("patch")...)
		}

		var installs []string
		done := make(map[string]struct{})
		for _, rule := range rules.goModules {
			for _, i := range rule.rule.AttrStrings("install") {
				if _, ok := done[i]; ok {
					continue
				}

				installs = append(installs, i)
				done[i] = struct{}{}
			}
		}

		// Check to see if the download rule was replacing an import path
		var dl *build.Rule
		for _, downloadRule := range rules.downloads {
			m := downloadRule.rule.AttrString("module")
			if m != modName {
				dl = downloadRule.rule
				break
			}
		}

		name := ""
		var aliases []labels.Label
		if len(rules.goModules) == 1 && rules.goModules[0].pkg == m.thirdPartyFolder {
			name = rules.goModules[0].rule.Name()
		} else {
			for _, goMod := range rules.goModules {
				aliases = append(aliases, labels.Label{Package: goMod.pkg, Target: goMod.rule.Name()})
			}
		}

		m.mods[modName] = &module{
			name:       name,
			moduleName: modName,
			download:   dl,
			version:    version,
			install:    installs,
			aliases:    aliases,
			patch:      patches,
		}
	}
}

func newGoRepoRule(module, version, download, name string, install, patches []string) *build.CallExpr {
	expr := &build.CallExpr{
		X: &build.Ident{Name: "go_repo"},
		List: []build.Expr{
			edit.NewAssignExpr("module", edit.NewStringExpr(module)),
			edit.NewAssignExpr("install", edit.NewStringList(install)),
		},
	}
	if version != "" {
		expr.List = append(expr.List, edit.NewAssignExpr("version", edit.NewStringExpr(version)))
	}
	if download != "" {
		expr.List = append(expr.List, edit.NewAssignExpr("download", edit.NewStringExpr(download)))
	}
	if name != "" {
		expr.List = append(expr.List, edit.NewAssignExpr("name", edit.NewStringExpr(name)))
	}
	if len(patches) != 0 {
		expr.List = append(expr.List, edit.NewAssignExpr("patch", edit.NewStringList(patches)))
	}
	return expr
}

func newAlias(name, module, thirdPartyDir string) *build.CallExpr {
	subrepoName := generate.SubrepoName(module, thirdPartyDir)
	install := generate.BuildTarget("installs", ".", subrepoName)

	rule := edit.NewRuleExpr("filegroup", name)
	rule.SetAttr("exported_deps", edit.NewStringList([]string{install}))
	rule.SetAttr("visibility", edit.NewStringList([]string{"PUBLIC"}))
	return rule.Call
}

// readModuleRules reads all the module rules from all the files and stores them in a single
func (m *Migrate) readModuleRules(f *build.File, pkg string) error {
	for _, rule := range f.Rules("go_module") {
		moduleName := rule.AttrString("module")

		mod, ok := m.moduleRules[moduleName]
		if !ok {
			mod = &moduleParts{}
			m.moduleRules[moduleName] = mod
		}

		// Add the download rule to the rules if it's not already there
		if dl := rule.AttrString("download"); dl != "" {
			l := labels.ParseRelative(dl, pkg)
			dlFile, err := m.graph.LoadFile(l.Package)
			if err != nil {
				return err
			}

			dlRule := graph.FindTargetByName(dlFile, l.Target)

			done := false
			for _, exitingDl := range mod.downloads {
				if exitingDl.rule == dlRule {
					done = true
				}
			}
			if !done {
				mod.downloads = append(mod.downloads, &targetRule{pkg: l.Package, rule: dlRule})
			}
		}

		mod.goModules = append(mod.goModules, &targetRule{pkg: pkg, rule: rule})
	}
	return nil
}
