package edit

import (
	"github.com/please-build/buildtools/build"

	"github.com/please-build/puku/kinds"
)

type Rule struct {
	Dir  string
	Kind *kinds.Kind
	*build.Rule
}

// SetOrDeleteAttr will make sure the attribute with the given name matches the values passed in. It will keep the
// existing expressions in the list to maintain things like comments.
func (rule *Rule) SetOrDeleteAttr(name string, values []string) {
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
			exprs = append(exprs, NewStringExpr(v))
		}
	}

	listExpr.List = exprs
	rule.SetAttr(name, listExpr)
}

func (rule *Rule) IsTest() bool {
	return rule.Kind.Type == kinds.Test
}

func (rule *Rule) SrcsAttr() string {
	return rule.Kind.SrcsAttr
}

func (rule *Rule) AddSrc(src string) {
	srcsAttr := rule.SrcsAttr()
	srcs := rule.AttrStrings(srcsAttr)
	rule.SetOrDeleteAttr(srcsAttr, append(srcs, src))
}

func (rule *Rule) RemoveSrc(rem string) {
	srcsAttr := rule.SrcsAttr()
	srcs := rule.AttrStrings(srcsAttr)
	set := make([]string, 0, len(srcs))
	for _, src := range srcs {
		if src != rem {
			set = append(set, src)
		}
	}
	rule.SetOrDeleteAttr(srcsAttr, set)
}

func (rule *Rule) LocalLabel() string {
	return ":" + rule.Name()
}

func (rule *Rule) Label() string {
	return BuildTarget(rule.Name(), rule.Dir, "")
}

func NewRule(r *build.Rule, kindType *kinds.Kind, pkgDir string) *Rule {
	return &Rule{
		Dir:  pkgDir,
		Kind: kindType,
		Rule: r,
	}
}
