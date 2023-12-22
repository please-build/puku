package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/please-build/buildtools/build"
	"github.com/please-build/buildtools/labels"
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
		moduleRules:      map[string]*moduleParts{},
	}
}

// Migrate replaces go_module rules with the equivalent go_repo rules, generating filegroup replacePartsWithAliases where appropriate
type Migrate struct {
	graph            *graph.Graph
	thirdPartyFolder string
	moduleRules      map[string]*moduleParts
}

// pkgRule represents the rule expr in a pkg
type pkgRule struct {
	pkg  string
	rule *build.Rule
}

type moduleParts struct {
	module string
	// The go_mod_download for this module, if any
	download *pkgRule
	// Any go_module rule(s) that compile the module
	parts []*pkgRule
	// Any go_module rules that have "binary = True"
	binaryParts []*pkgRule
}

func isInternal(path string) bool {
	for _, p := range strings.Split(path, string(filepath.Separator)) {
		if p == "internal" {
			return true
		}
	}
	return false
}

func (p *moduleParts) installs() []string {
	var installs []string
	done := make(map[string]struct{})
	for _, part := range p.parts {
		is := part.rule.AttrStrings("install")
		if len(is) == 0 {
			is = []string{"."}
		}

		for _, i := range is {
			// We don't need to install internal things anymore with go_repo.
			if isInternal(i) {
				continue
			}
			if _, ok := done[i]; !ok {
				installs = append(installs, i)
				done[i] = struct{}{}
			}
		}
	}
	return installs
}

// deps returns the dependencies of the module parts
func (p *moduleParts) deps() []string {
	var deps []string
	done := make(map[string]struct{})
	for _, part := range p.parts {
		ds := part.rule.AttrStrings("deps")
		for _, dep := range ds {
			if _, ok := done[dep]; !ok {
				deps = append(deps, dep)
				done[dep] = struct{}{}
			}
		}
	}
	return deps
}

func binaryAlias(module, thirdPartyDir string, part *pkgRule) (*build.Rule, error) {
	rule := edit.NewRuleExpr("filegroup", part.rule.Name())
	rule.SetAttr("binary", &build.Ident{Name: "True"})

	installs := part.rule.AttrStrings("install")

	if len(installs) == 0 {
		rule.SetAttr("srcs", edit.NewStringList([]string{generate.SubrepoTarget(module, thirdPartyDir, "")}))
	} else if len(installs) == 1 {
		rule.SetAttr("srcs", edit.NewStringList([]string{generate.SubrepoTarget(module, thirdPartyDir, installs[0])}))
	} else {
		return nil, fmt.Errorf("too many installs to binary rule: %s", generate.BuildTarget(rule.Name(), part.pkg, ""))
	}

	return rule, nil
}

func (m *Migrate) Migrate(write bool, modules []string, paths ...string) error {
	// Read all the BUILD files under the provided paths to find go_module and go_mod_download rules
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
	}

	// Now we can generate all the rules we need
	if err := m.replaceRulesForModules(modules); err != nil {
		return err
	}
	return m.graph.FormatFiles(write, os.Stdout)
}

func (m *Migrate) replaceRulesForModules(modules []string) error {
	// If we're not migrating specific modules, do all of them
	if len(modules) == 0 {
		for _, parts := range m.moduleRules {
			if err := m.replaceRules(parts, nil); err != nil {
				return err
			}
		}
	}

	for _, mod := range modules {
		parts, ok := m.moduleRules[mod]
		if !ok {
			return fmt.Errorf("couldn't find go_module rules for %v", mod)
		}
		if err := m.replaceRules(parts, modules); err != nil {
			return err
		}
	}

	return nil
}

func ruleIdx(file *build.File, rule *build.Rule) int {
	for idx, expr := range file.Stmt {
		if expr == rule.Call {
			return idx
		}
	}
	return -1
}

func (m *Migrate) replaceRules(p *moduleParts, modsBeingMigrated []string) error {
	download := ""
	var version string
	var patches []string
	var licences []string
	var name = strings.ReplaceAll(p.module, "/", "_")

	thirdPartyFile, err := m.graph.LoadFile(m.thirdPartyFolder)
	if err != nil {
		return err
	}

	// We need to use a go_mod_download if the download rule is downloading the module using a different path than the
	// import path of the module e.g. for when we've forked a module similar to how replace works in go.mods.
	if p.download != nil && p.module != p.download.rule.AttrString("module") {
		download = labels.Shorten(generate.BuildTarget(p.download.rule.Name(), p.download.pkg, ""), m.thirdPartyFolder)
		if len(p.parts) == 1 {
			name = p.parts[0].rule.Name()
		}
	} else if p.download != nil {
		// Otherwise we don't need the download rule anymore
		downloadIdx := ruleIdx(thirdPartyFile, p.download.rule)
		thirdPartyFile.Stmt = append(thirdPartyFile.Stmt[:downloadIdx], thirdPartyFile.Stmt[downloadIdx+1:]...)
	}

	if p.download != nil {
		version = p.download.rule.AttrString("version")
		patches = p.download.rule.AttrStrings("patches")
		licences = p.download.rule.AttrStrings("licences")
	} else if len(p.parts) > 0 {
		if len(p.parts) == 1 {
			patches = p.parts[0].rule.AttrStrings("patches")
			licences = p.parts[0].rule.AttrStrings("licences")
		}
		for _, p := range p.parts {
			v := p.rule.AttrString("version")
			if version == "" || semver.Compare(version, v) < 0 {
				version = v
			}
		}
	} else {
		if len(p.binaryParts) == 1 {
			patches = p.binaryParts[0].rule.AttrStrings("patches")
		}
		for _, p := range p.binaryParts {
			v := p.rule.AttrString("version")
			if version == "" || semver.Compare(version, v) < 0 {
				version = v
			}
		}
	}

	// When we have just one part, and that part is in the third party folder, we don't need to use filegroups for
	// aliases. We can directly replace the module part with the go_repo rule.
	shouldReplaceFirstPartWithRepoRule := len(p.parts) == 1 && p.parts[0].pkg == m.thirdPartyFolder
	if shouldReplaceFirstPartWithRepoRule {
		name = p.parts[0].rule.Name()
	}

	deps, err := m.modDeps(p.deps(), modsBeingMigrated)
	if err != nil {
		return err
	}

	repoRule := newGoRepoRule(
		p.module,
		version,
		download,
		name,
		p.installs(),
		patches,
		deps,
		licences,
	)
	if shouldReplaceFirstPartWithRepoRule {
		idx := ruleIdx(thirdPartyFile, p.parts[0].rule)
		thirdPartyFile.Stmt[idx] = repoRule.Call
	} else {
		part := append(p.parts, p.binaryParts...)[0]
		if part.pkg == m.thirdPartyFolder {
			idx := ruleIdx(thirdPartyFile, part.rule)
			var stmts []build.Expr // Make sure this is a new slice otherwise we'll modify the underlying slice
			stmts = append(append(stmts, thirdPartyFile.Stmt[:idx]...), repoRule.Call)
			thirdPartyFile.Stmt = append(stmts, thirdPartyFile.Stmt[idx:]...)
		} else {
			thirdPartyFile.Stmt = append(thirdPartyFile.Stmt, repoRule.Call)
		}
	}

	if err := m.replacePartsWithAliases(p); err != nil {
		return err
	}

	return m.replaceBinaryWithAliases(p)
}

// modDeps returns any dependencies of this rule that are go_modules
func (m *Migrate) modDeps(deps, modsBeingMigrated []string) ([]string, error) {
	// If we don't pass any mods then we are migrating all modules so we shouldn't have any deps
	if len(modsBeingMigrated) == 0 {
		return nil, nil
	}

	modsInScope := make(map[string]struct{}, len(modsBeingMigrated))
	for _, mod := range modsBeingMigrated {
		modsInScope[mod] = struct{}{}
	}

	goModDeps := make([]string, 0, len(deps))
	for _, dep := range deps {
		label := labels.ParseRelative(dep, m.thirdPartyFolder)
		file, err := m.graph.LoadFile(label.Package)
		if err != nil {
			return nil, err
		}

		rule := edit.FindTargetByName(file, label.Target)
		if rule == nil {
			continue
		}
		if rule.Kind() == "go_module" {
			// Check if this guy is going to be rewritten as a go_repo by the end of this
			if _, ok := modsInScope[rule.AttrString("module")]; ok {
				continue
			}
			goModDeps = append(goModDeps, label.FormatRelative(m.thirdPartyFolder))
		}
	}
	return goModDeps, nil
}

func (m *Migrate) replacePartsWithAliases(p *moduleParts) error {
	if len(p.parts) == 1 && p.parts[0].pkg == m.thirdPartyFolder {
		return nil
	}
	for _, part := range p.parts {
		subrepoName := generate.SubrepoName(p.module, m.thirdPartyFolder)
		installRule := generate.BuildTarget("installs", ".", subrepoName)

		rule := edit.NewRuleExpr("filegroup", part.rule.Name())

		// Just export the whole set of installs. We can't do much better without trying to parse and evaluate the
		// wildcards (i.e. "pkg/...") ourselves.
		rule.SetAttr("exported_deps", edit.NewStringList([]string{installRule}))

		f, err := m.graph.LoadFile(part.pkg)
		if err != nil {
			return err
		}

		f.Stmt[ruleIdx(f, part.rule)] = rule.Call
	}
	return nil
}

func (m *Migrate) replaceBinaryWithAliases(p *moduleParts) error {
	for _, part := range p.binaryParts {
		rule, err := binaryAlias(p.module, m.thirdPartyFolder, part)
		if err != nil {
			return err
		}
		f, err := m.graph.LoadFile(part.pkg)
		if err != nil {
			return err
		}
		idx := ruleIdx(f, part.rule)
		f.Stmt[idx] = rule.Call
	}
	return nil
}

func newGoRepoRule(module, version, download, name string, install, patches, deps []string, licences []string) *build.Rule {
	expr := &build.CallExpr{
		X: &build.Ident{Name: "go_repo"},
		List: []build.Expr{
			edit.NewAssignExpr("module", edit.NewStringExpr(module)),
		},
	}
	if name != "" {
		expr.List = append(expr.List, edit.NewAssignExpr("name", edit.NewStringExpr(name)))
	}
	if len(install) != 0 {
		expr.List = append(expr.List, edit.NewAssignExpr("install", edit.NewStringList(install)))
	}
	if len(deps) != 0 {
		expr.List = append(expr.List, edit.NewAssignExpr("deps", edit.NewStringList(deps)))
	}
	if len(licences) != 0 {
		expr.List = append(expr.List, edit.NewAssignExpr("licences", edit.NewStringList(licences)))
	}

	if download != "" {
		expr.List = append(expr.List, edit.NewAssignExpr("download", edit.NewStringExpr(download)))
	} else {
		if version != "" {
			expr.List = append(expr.List, edit.NewAssignExpr("version", edit.NewStringExpr(version)))
		}
		if len(patches) != 0 {
			expr.List = append(expr.List, edit.NewAssignExpr("patch", edit.NewStringList(patches)))
		}
	}

	return build.NewRule(expr)
}

// readModuleRules reads all the module rules from all the files and stores them in a single
func (m *Migrate) readModuleRules(f *build.File, pkg string) error {
	for _, rule := range f.Rules("go_module") {
		moduleName := rule.AttrString("module")

		mod, ok := m.moduleRules[moduleName]
		if !ok {
			mod = &moduleParts{
				module: moduleName,
			}
			m.moduleRules[moduleName] = mod
		}

		// Add the download rule to the rules if it's not already there
		if dl := rule.AttrString("download"); dl != "" {
			l := labels.ParseRelative(dl, pkg)
			dlFile, err := m.graph.LoadFile(l.Package)
			if err != nil {
				return err
			}

			dlRule := edit.FindTargetByName(dlFile, l.Target)
			if dlRule == nil {
				return fmt.Errorf("failed to find :%v referrenced by :%v", l.Target, rule.Name())
			}

			if mod.download == nil {
				mod.download = &pkgRule{pkg: l.Package, rule: dlRule}
			} else {
				existingVer := mod.download.rule.AttrString("version")
				newVer := dlRule.AttrString("version")
				if semver.Compare(existingVer, newVer) < 0 {
					mod.download = &pkgRule{pkg: l.Package, rule: dlRule}
				}
			}
		}

		if edit.BoolAttr(rule, "binary") {
			mod.binaryParts = append(mod.binaryParts, &pkgRule{pkg: pkg, rule: rule})
		} else {
			mod.parts = append(mod.parts, &pkgRule{pkg: pkg, rule: rule})
		}
	}
	return nil
}
