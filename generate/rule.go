package generate

import (
	"github.com/bazelbuild/buildtools/build"

	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/glob"
	"github.com/please-build/puku/kinds"
)

type rule struct {
	dir  string
	kind *kinds.Kind
	*build.Rule
}

func (rule *rule) parseGlob() *glob.GlobArgs {
	srcs := rule.Attr("srcs")
	if srcs == nil {
		return nil
	}

	call, ok := srcs.(*build.CallExpr)
	if !ok {
		return nil
	}

	ident, ok := call.X.(*build.Ident)
	if !ok {
		return nil
	}

	if ident.Name != "glob" {
		return nil
	}

	var include, exclude []string
	positionalPos := 0
	for _, expr := range call.List {
		assign, ok := expr.(*build.AssignExpr)
		if ok {
			ident := assign.LHS.(*build.Ident)
			if !ok {
				return nil
			}
			if ident.Name == "include" {
				include = build.Strings(assign.RHS)
			}
			if ident.Name == "exclude" {
				exclude = build.Strings(assign.RHS)
			}
			continue // ignore other args. We don't care about include_symlinks etc.
		}

		if positionalPos == 0 {
			include = build.Strings(expr)
		}
		if positionalPos == 1 {
			exclude = build.Strings(expr)
		}
		positionalPos++
	}
	return &glob.GlobArgs{
		Include: include,
		Exclude: exclude,
	}
}

func (rule *rule) setOrDeleteAttr(name string, values []string) {
	if len(values) == 0 {
		rule.DelAttr(name)
		return
	}
	rule.SetAttr(name, edit.NewStringList(values))
}

func (rule *rule) isTest() bool {
	return rule.kind.Type == kinds.Test
}

func (rule *rule) addSrc(src string) {
	srcs := rule.AttrStrings("srcs")
	rule.setOrDeleteAttr("srcs", append(srcs, src))
}

func (rule *rule) removeSrc(rem string) {
	srcs := rule.AttrStrings("srcs")
	set := make([]string, 0, len(srcs))
	for _, src := range srcs {
		if src != rem {
			set = append(set, src)
		}
	}
	rule.setOrDeleteAttr("srcs", set)
}

func (rule *rule) setExternal() {
	rule.SetAttr("external", &build.Ident{Name: "True"})
}

func (rule *rule) localLabel() string {
	return ":" + rule.Name()
}

func (rule *rule) label() string {
	return buildTarget(rule.Name(), rule.dir, "")
}

func (rule *rule) isExternal() bool {
	if !rule.isTest() {
		return false
	}

	external := rule.Attr("external")
	if external == nil {
		return false
	}

	ident, ok := external.(*build.Ident)
	if !ok {
		return false
	}

	return ident.Name == "True"
}

func newRule(r *build.Rule, kindType *kinds.Kind, pkgDir string) *rule {
	return &rule{
		dir:  pkgDir,
		kind: kindType,
		Rule: r,
	}
}
