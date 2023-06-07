package generate

import (
	"fmt"
	"github.com/bazelbuild/buildtools/build"
	"github.com/please-build/puku/glob"
)

func newStringExpr(s string) *build.StringExpr {
	return &build.StringExpr{Value: s}
}

func newStringList(ss []string) *build.ListExpr {
	l := new(build.ListExpr)
	for _, s := range ss {
		l.List = append(l.List, newStringExpr(s))
	}
	return l
}

type rule struct {
	dir      string
	kindType KindType
	*build.Rule
}

func (r *rule) allSources() ([]string, error) {
	if call, ok := r.Attr("srcs").(*build.CallExpr); ok {
		srcs, err := evalGlob(r.dir, call)
		if err != nil {
			return nil, fmt.Errorf("failed to eval glob in %v: %v", r.dir, err)
		}

		return srcs, nil
	}

	return r.AttrStrings("srcs"), nil
}

func evalGlob(dir string, call *build.CallExpr) ([]string, error) {
	if i, ok := call.X.(*build.Ident); !ok || i.Name != "glob" {
		return nil, nil
	}

	var include, exclude []string
	positionalPos := 0
	for _, expr := range call.List {
		assign, ok := expr.(*build.AssignExpr)
		if ok {
			ident := assign.LHS.(*build.Ident)
			if !ok {
				return nil, nil
			}
			if ident.Name == "include" {
				include = build.Strings(assign.RHS)
				continue
			}
			if ident.Name == "exclude" {
				exclude = build.Strings(assign.RHS)
				continue
			}
		}

		if positionalPos == 0 {
			include = build.Strings(expr)
		}
		if positionalPos == 1 {
			exclude = build.Strings(expr)
		}
		positionalPos++
	}
	return glob.Glob(dir, include, exclude)
}

func (r *rule) setOrDeleteAttr(name string, values []string) {
	if len(values) == 0 {
		r.DelAttr(name)
		return
	}
	r.SetAttr(name, newStringList(values))
}

func (r *rule) isTest() bool {
	return r.kindType == KindType_Test
}

func (r *rule) addSrc(src string) {
	srcs := r.AttrStrings("srcs")
	r.setOrDeleteAttr("srcs", append(srcs, src))
}

func (r *rule) setExternal() {
	r.SetAttr("external", &build.Ident{Name: "True"})
}

func newRule(r *build.Rule, kindType KindType, pkgDir string) *rule {
	return &rule{
		dir:      pkgDir,
		kindType: kindType,
		Rule:     r,
	}
}
