package generate

import (
	"github.com/please-build/buildtools/build"

	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/kinds"
)

type rule struct {
	dir  string
	kind *kinds.Kind
	*build.Rule
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
	return BuildTarget(rule.Name(), rule.dir, "")
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
