package edit

import (
	"github.com/please-build/buildtools/build"
	"github.com/please-build/buildtools/edit"
)

func EnsureSubinclude(file *build.File) {
	var subinclude *build.CallExpr
	for _, expr := range file.Stmt {
		call, ok := expr.(*build.CallExpr)
		if !ok {
			continue
		}

		x, ok := call.X.(*build.Ident)
		if !ok {
			continue
		}

		if x.Name != "subinclude" {
			continue
		}
		if subinclude == nil {
			subinclude = call
		}

		for _, inc := range call.List {
			str, ok := inc.(*build.StringExpr)
			if !ok {
				continue
			}

			if str.Value == "///go//build_defs:go" {
				return
			}
		}
	}
	if subinclude == nil {
		subinclude = &build.CallExpr{
			X: &build.Ident{Name: "subinclude"},
		}
		file.Stmt = append([]build.Expr{subinclude}, file.Stmt...)
	}
	subinclude.List = append(subinclude.List, NewStringExpr("///go//build_defs:go"))
}

func FindTargetByName(file *build.File, name string) *build.Rule {
	for _, rule := range file.Rules("") {
		if rule.Name() == name {
			return rule
		}
	}
	return nil
}

// RemoveTarget removes the target with the given name from the build file
func RemoveTarget(file *build.File, rule *build.Rule) bool {
	for i, r := range file.Rules("") {
		// Compare by the call expression as the rule created in Rules will not match
		if r.Call != rule.Call {
			continue
		}

		file.Stmt = append(file.Stmt[:i], file.Stmt[(i+1):]...)
		return true
	}
	return false
}

func NewStringExpr(s string) *build.StringExpr {
	return &build.StringExpr{Value: s}
}

func NewAssignExpr(key string, value build.Expr) *build.AssignExpr {
	return &build.AssignExpr{
		LHS: &build.Ident{Name: key},
		Op:  "=",
		RHS: value,
	}
}

func NewStringList(ss []string) *build.ListExpr {
	l := new(build.ListExpr)
	for _, s := range ss {
		l.List = append(l.List, NewStringExpr(s))
	}
	return l
}

func NewRuleExpr(kind, name string) *build.Rule {
	rule, _ := edit.ExprToRule(&build.CallExpr{
		X:    &build.Ident{Name: kind},
		List: []build.Expr{},
	}, kind)

	if name != "" {
		rule.SetAttr("name", NewStringExpr(name))
	}

	return rule
}

func BoolAttr(rule *build.Rule, attrName string) bool {
	attr := rule.Attr(attrName)
	if attr == nil {
		return false
	}

	ident, ok := attr.(*build.Ident)
	if !ok {
		return false
	}

	return ident.Name == "True"
}
