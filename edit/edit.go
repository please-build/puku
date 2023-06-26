package edit

import (
	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/edit"
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
