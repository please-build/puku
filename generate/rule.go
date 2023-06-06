package generate

import (
	"strings"

	"github.com/bazelbuild/buildtools/build"
)

type Rule struct {
	name          string
	kind          string
	srcs          []string
	cgoSrcs       []string
	compilerFlags []string
	linkerFlags   []string
	pkgConfigs    []string
	asmFiles      []string
	hdrs          []string
	deps          []string
	embedPatterns []string
	test, external bool
}

func NewStringExpr(s string) *build.StringExpr {
	return &build.StringExpr{Value: s}
}

func NewStringList(ss []string) *build.ListExpr {
	l := new(build.ListExpr)
	for _, s := range ss {
		l.List = append(l.List, NewStringExpr(s))
	}
	return l
}

func SetOrDeleteAttr(r *build.Rule, name string, values []string) {
	if len(values) == 0 {
		r.DelAttr(name)
		return
	}
	r.SetAttr(name, NewStringList(values))
}

func populateRule(r *build.Rule, targetState *Rule) {
	r.SetKind(targetState.kind)
	r.SetAttr("name", NewStringExpr(targetState.name))

	if len(targetState.cgoSrcs) > 0 {
		SetOrDeleteAttr(r, "srcs", targetState.cgoSrcs)
		SetOrDeleteAttr(r, "go_srcs", targetState.srcs)
	} else {
		SetOrDeleteAttr(r, "srcs", targetState.srcs)
	}

	SetOrDeleteAttr(r, "deps", targetState.deps)
	SetOrDeleteAttr(r, "pkg_config", targetState.pkgConfigs)
	SetOrDeleteAttr(r, "compiler_flags", targetState.compilerFlags)
	SetOrDeleteAttr(r, "linker_flags", targetState.linkerFlags)
	SetOrDeleteAttr(r, "hdrs", targetState.hdrs)
	SetOrDeleteAttr(r, "asm_srcs", targetState.asmFiles)

	if len(targetState.embedPatterns) > 0 {
		r.SetAttr("resources", &build.CallExpr{
			X: &build.Ident{Name: "glob"},
			List: []build.Expr{
				NewStringList(targetState.embedPatterns),
			},
		})
	} else {
		r.DelAttr("resources")
	}
}

func callToRule(call *build.CallExpr) *Rule {
	rule := build.NewRule(call)
	ret := &Rule{name: rule.Name(), kind: rule.Kind()}
	cgo := strings.HasPrefix(rule.Kind(), "cgo_")
	ret.test = strings.HasSuffix(rule.Kind(), "_test")

	if cgo {
		ret.srcs = rule.AttrStrings("go_srcs")
		ret.cgoSrcs = rule.AttrStrings("srcs")
	} else {
		ret.srcs = rule.AttrStrings("srcs")
	}

	ret.deps = rule.AttrStrings("deps")
	ret.deps = rule.AttrStrings("pkg_config")
	ret.deps = rule.AttrStrings("compiler_flags")
	ret.deps = rule.AttrStrings("linker_flags")
	ret.deps = rule.AttrStrings("hdrs")
	ret.deps = rule.AttrStrings("asm_srcs")
	return ret
}
