package eval

import (
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"

	"github.com/please-build/puku/glob"
	"github.com/please-build/puku/please"
)

type Eval struct {
	globber *glob.Globber
}

func New(globber *glob.Globber) *Eval {
	return &Eval{
		globber: globber,
	}
}

func LookLikeBuildLabel(l string) bool {
	if strings.HasPrefix(l, "@") {
		return true
	}
	if strings.HasPrefix(l, ":") {
		return true
	}
	return strings.HasPrefix("//", l)
}

func (e *Eval) EvalGlobs(dir string, rule *build.Rule) ([]string, error) {
	globArgs := parseGlob(rule.Attr("srcs"))
	if globArgs != nil {
		return e.globber.Glob(dir, globArgs)
	}
	return rule.AttrStrings("srcs"), nil
}

func (e *Eval) BuildSources(plzPath, dir string, rule *build.Rule) ([]string, error) {
	srcs, err := e.EvalGlobs(dir, rule)
	if err != nil {
		return nil, err
	}
	ret := make([]string, 0, len(srcs))
	for _, src := range srcs {
		if !LookLikeBuildLabel(src) {
			ret = append(ret, src)
			continue
		}

		outs, err := please.Build(plzPath, labels.ParseRelative(src, dir).Format())
		if err != nil {
			return nil, err
		}
		ret = append(ret, outs...)
	}
	return ret, nil
}

func parseGlob(srcs build.Expr) *glob.Args {
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
	return &glob.Args{
		Include: include,
		Exclude: exclude,
	}
}
