package generate

import (
	"bytes"
	"text/template"

	"github.com/please-build/buildtools/build"

	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/kinds"
)

type rule struct {
	dir  string
	kind *kinds.Kind
	*build.Rule
	SrcsOveride build.Expr
}

// setOrDeleteAttr will make sure the attribute with the given name matches the values passed in. It will keep the
// existing expressions in the list to maintain things like comments.
func (rule *rule) setOrDeleteAttr(name string, values []string) {
	if len(values) == 0 {
		rule.DelAttr(name)
		return
	}

	valuesMap := make(map[string]struct{})
	for _, v := range values {
		valuesMap[v] = struct{}{}
	}

	listExpr, _ := rule.Attr(name).(*build.ListExpr)
	if listExpr == nil {
		listExpr = &build.ListExpr{}
	}

	exprs := make([]build.Expr, 0, len(values))
	done := map[string]struct{}{}

	// Loop through the existing values, filtering out any that aren't supposed to be there
	for _, expr := range listExpr.List {
		val, ok := expr.(*build.StringExpr)
		if !ok {
			continue
		}
		if _, ok := valuesMap[val.Value]; ok {
			exprs = append(exprs, val)
			done[val.Value] = struct{}{}
		}
	}

	// Loops through the value adding any new values that didn't used to be there
	for _, v := range values {
		if _, done := done[v]; !done {
			exprs = append(exprs, edit.NewStringExpr(v))
		}
	}

	listExpr.List = exprs
	rule.SetAttr(name, listExpr)
}

func (rule *rule) isTest() bool {
	return rule.kind.Type == kinds.Test
}

func (rule *rule) SrcsAttr() string {
	return rule.kind.SrcsAttr
}

func (rule *rule) addSrc(src string) {
	if rule.SrcsOveride != nil {
		return
	}
	srcsAttr := rule.SrcsAttr()
	srcs := rule.AttrStrings(srcsAttr)
	rule.setOrDeleteAttr(srcsAttr, append(srcs, src))
}

func (rule *rule) removeSrc(rem string) {
	if rule.SrcsOveride != nil {
		return
	}
	srcsAttr := rule.SrcsAttr()
	srcs := rule.AttrStrings(srcsAttr)
	set := make([]string, 0, len(srcs))
	for _, src := range srcs {
		if src != rem {
			set = append(set, src)
		}
	}
	rule.setOrDeleteAttr(srcsAttr, set)
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
		dir:         pkgDir,
		kind:        kindType,
		Rule:        r,
		SrcsOveride: templateSrcs(kindType.SrcsRuleTemplate, r),
	}
}

func (rule *rule) Srcs() build.Expr {
	if rule.SrcsOveride != nil {
		return rule.SrcsOveride
	}
	return rule.Attr(rule.SrcsAttr())
}

func templateSrcs(templStr string, r *build.Rule) build.Expr {
	if templStr == "" {
		return nil
	}
	tmpl, err := template.New("test").Parse(templStr)
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, r)
	if err != nil {
		return nil
	}
	file, err := build.ParseBuild("template", buf.Bytes())
	if err != nil {
		return nil
	}
	if len(file.Stmt) != 1 {
		return nil
	}
	return file.Stmt[0]
}
