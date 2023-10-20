package migrate

import (
	"fmt"
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
		moduleRules:      map[string]*moduleParts{},
	}
}

// Migrate replaces go_module rules with the equivalent go_repo rules, generating filegroup writeAliases where appropriate
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

func (p *moduleParts) writeRules(thirdPartyDir string, g *graph.Graph) error {
	download := ""
	var version string
	var patches []string
	var name = strings.ReplaceAll(p.module, "/", "_")

	thirdPartyFile, err := g.LoadFile(thirdPartyDir)
	if err != nil {
		return err
	}

	// We need to use a go_mod_download if the download rule is downloading the module using a different path than the
	// import path of the module e.g. for when we've forked a module similar to how replace works in go.mods.
	if p.download != nil && p.module != p.download.rule.AttrString("module") {
		download = labels.Shorten(generate.BuildTarget(p.download.rule.Name(), p.download.pkg, ""), thirdPartyDir)
		// Add the download rule back in as we still need this
		thirdPartyFile.Stmt = append(thirdPartyFile.Stmt, p.download.rule.Call)
		if len(p.parts) == 1 {
			name = p.parts[0].rule.Name()
		}
	}

	if p.download != nil {
		version = p.download.rule.AttrString("version")
		patches = p.download.rule.AttrStrings("patches")
	} else if len(p.parts) > 0 {
		if len(p.parts) == 1 {
			patches = p.parts[0].rule.AttrStrings("patches")
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

	if len(p.parts) == 1 && p.parts[0].pkg == thirdPartyDir {
		name = p.parts[0].rule.Name()
	}

	thirdPartyFile.Stmt = append(thirdPartyFile.Stmt, newGoRepoRule(
		p.module,
		version,
		download,
		name,
		p.installs(),
		patches,
	).Call)

	if err := p.writeAliases(thirdPartyDir, g); err != nil {
		return err
	}

	return p.writeBinaryAliases(thirdPartyDir, g)
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
			if _, ok := done[i]; !ok {
				installs = append(installs, i)
				done[i] = struct{}{}
			}
		}
	}
	return installs
}

func (p *moduleParts) writeAliases(thirdPartyDir string, g *graph.Graph) error {
	if len(p.parts) == 1 && p.parts[0].pkg == thirdPartyDir {
		return nil
	}
	for _, part := range p.parts {
		subrepoName := generate.SubrepoName(p.module, thirdPartyDir)
		installRule := generate.BuildTarget("installs", ".", subrepoName)

		rule := edit.NewRuleExpr("filegroup", part.rule.Name())

		// Just export the whole set of installs. We can't do much better without trying to parse and evaluate the
		// wildcards (i.e. "pkg/...") ourselves.
		rule.SetAttr("exported_deps", edit.NewStringList([]string{installRule}))

		f, err := g.LoadFile(part.pkg)
		if err != nil {
			return err
		}

		f.Stmt = append(f.Stmt, rule.Call)
	}
	return nil
}

func (p *moduleParts) writeBinaryAliases(thirdPartyDir string, g *graph.Graph) error {
	for _, part := range p.binaryParts {
		rule, err := binaryAlias(p.module, thirdPartyDir, part)
		if err != nil {
			return err
		}
		f, err := g.LoadFile(part.pkg)
		if err != nil {
			return err
		}

		f.Stmt = append(f.Stmt, rule.Call)
	}
	return nil
}

func binaryAlias(module, thirdPartyDir string, part *pkgRule) (*build.Rule, error) {
	rule := edit.NewRuleExpr("filegroup", part.rule.Name())
	rule.SetAttr("binary", &build.Ident{Name: "True"})

	installs := part.rule.AttrStrings("install")

	if len(installs) == 0 {
		rule.SetAttr("exported_deps", edit.NewStringList([]string{generate.SubrepoTarget(module, thirdPartyDir, "")}))
	} else if len(installs) == 1 {
		rule.SetAttr("exported_deps", edit.NewStringList([]string{generate.SubrepoTarget(module, thirdPartyDir, installs[0])}))
	} else {
		return nil, fmt.Errorf("too many installs to binary rule: %s", generate.BuildTarget(rule.Name(), part.pkg, ""))
	}

	return rule, nil
}

func (m *Migrate) Migrate(write bool, paths ...string) error {
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

	// After loading all the rules, we can delete the old repo rules, as we're going to replace them now.
	for _, path := range paths {
		f, err := m.graph.LoadFile(path)
		if err != nil {
			return err
		}

		// Delete all the rules from the files
		f.DelRules("go_module", "")
		f.DelRules("go_mod_download", "")
	}

	// Now we can generate all the rules we need
	if err := m.genRules(); err != nil {
		return err
	}
	return m.graph.FormatFiles(write, os.Stdout)
}

func (m *Migrate) genRules() error {
	for _, parts := range m.moduleRules {
		if err := parts.writeRules(m.thirdPartyFolder, m.graph); err != nil {
			return err
		}
	}

	return nil
}

func newGoRepoRule(module, version, download, name string, install, patches []string) *build.Rule {
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

			dlRule := graph.FindTargetByName(dlFile, l.Target)
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
