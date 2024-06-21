package edit

import (
	"errors"
	"strings"

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

func NewGoRepoRule(mod, version, download string, licences []string, labels []string) *build.CallExpr {
	rule := build.NewRule(&build.CallExpr{
		X:    &build.Ident{Name: "go_repo"},
		List: []build.Expr{},
	})
	rule.SetAttr("module", NewStringExpr(mod))
	if version != "" {
		rule.SetAttr("version", NewStringExpr(version))
	}
	if download != "" {
		rule.SetAttr("download", NewStringExpr(":"+download))
	}
	if len(licences) != 0 {
		rule.SetAttr("licences", NewStringList(licences))
	}
	if len(labels) != 0 {
		rule.SetAttr("labels", NewStringList(labels))
	}
	return rule.Call
}

func NewModDownloadRule(mod, version string, licences []string) (*build.CallExpr, string) {
	rule := NewRuleExpr("go_mod_download", strings.ReplaceAll(mod, "/", "_")+"_dl")

	rule.SetAttr("module", NewStringExpr(mod))
	rule.SetAttr("version", NewStringExpr(version))
	if len(licences) != 0 {
		rule.SetAttr("licences", NewStringList(licences))
	}
	return rule.Call, rule.Name()
}

// AddLabel adds a specified string label to a build Rule's labels, unless it already exists
func AddLabel(rule *build.Rule, label string) error {
	// Fetch the labels attribute, or initialise it
	ruleLabels := rule.Attr("labels")
	if ruleLabels == nil {
		ruleLabels = &build.ListExpr{}
	}
	// Check it's a list of expressions
	ruleLabelsList, ok := ruleLabels.(*build.ListExpr)
	if !ok {
		return errors.New("rule already has a `labels` attribute, and it is not a list")
	}
	// Check for already-matching label
	for _, labelExpr := range ruleLabelsList.List {
		// Ignore any non-string labels
		labelStringExpr, ok := labelExpr.(*build.StringExpr)
		if !ok {
			continue
		}
		// If a matching label already exists, no need to do anything
		if labelStringExpr.Value == label {
			return nil
		}
	}
	// Add the new label
	ruleLabelsList.List = append(ruleLabelsList.List, NewStringExpr(label))
	return nil
}
