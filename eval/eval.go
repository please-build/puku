package eval

import (
	"strings"

	"github.com/please-build/buildtools/build"
	"github.com/please-build/buildtools/labels"

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

func (e *Eval) EvalGlobs(dir string, rule *build.Rule, attrName string) ([]string, error) {
	files, err := e.evalGlobs(dir, rule.Attr(attrName))
	if err != nil {
		return nil, err
	}
	ret := make([]string, 0, len(files))
	for f := range files {
		ret = append(ret, f)
	}
	return ret, nil
}

func (e *Eval) evalGlobs(dir string, val build.Expr) (map[string]struct{}, error) {
	switch expr := val.(type) {
	case *build.CallExpr:
		globArgs := parseGlob(expr)
		if globArgs == nil {
			return nil, nil
		}
		return e.globber.Glob(dir, globArgs)
	case *build.BinaryExpr:
		ret, err := e.evalGlobs(dir, expr.X)
		if err != nil {
			return nil, err
		}
		y, err := e.evalGlobs(dir, expr.Y)
		if err != nil {
			return nil, err
		}
		switch expr.Op {
		case "+":
			for f := range y {
				ret[f] = struct{}{}
			}
		case "-":
			for f := range y {
				delete(ret, f)
			}
		}
		return ret, nil
	default:
		str := build.Strings(expr)
		ret := make(map[string]struct{}, len(str))
		for _, s := range str {
			ret[s] = struct{}{}
		}
		return ret, nil
	}
}

func (e *Eval) BuildSources(plzPath, dir string, rule *build.Rule, srcsArg string) ([]string, error) {
	srcs, err := e.EvalGlobs(dir, rule, srcsArg)
	if err != nil {
		return nil, err
	}
	ret := make([]string, 0, len(srcs))
	for _, src := range srcs {
		if !LookLikeBuildLabel(src) {
			ret = append(ret, src)
			continue
		}
		target, err := please.RecursivelyProvide(plzPath, labels.ParseRelative(src, dir).Format(), "go")
		if err != nil {
			return nil, err
		}

		outs, err := please.Build(plzPath, target)
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
