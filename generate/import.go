package generate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/please-build/puku/kinds"
)

// GoFile represents a single Go file in a package
type GoFile struct {
	// Name is the name from the package clause of this file
	Name, FileName string
	// Imports are the imports of this file
	Imports []string
}

// ImportDir does _some_ of what the go/build ImportDir does but is more permissive.
func ImportDir(dir string) (map[string]*GoFile, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*GoFile, len(files))
	for _, info := range files {
		if info.IsDir() {
			continue
		}

		if filepath.Ext(info.Name()) != ".go" {
			continue
		}

		f, err := importFile(dir, info.Name())
		if err != nil {
			return nil, err
		}

		if f != nil {
			ret[info.Name()] = f
		}
	}

	return ret, nil
}

func importFile(dir, src string) (*GoFile, error) {
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, src), nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if IsGenerated(f) {
		return nil, nil
	}
	imports := make([]string, 0, len(f.Imports))
	for _, i := range f.Imports {
		path := i.Path.Value
		path = strings.Trim(path, `"`)
		imports = append(imports, path)
	}

	return &GoFile{
		Name:     f.Name.Name,
		FileName: src,
		Imports:  imports,
	}, nil
}

// IsExternal returns whether the test is external
func (f *GoFile) IsExternal(pkgName string) bool {
	return f.Name == filepath.Base(pkgName)+"_test" && f.IsTest()
}

func (f *GoFile) IsTest() bool {
	return strings.HasSuffix(f.FileName, "_test.go")
}

func (f *GoFile) IsCmd() bool {
	return f.Name == "main"
}

func (f *GoFile) kindType() kinds.Type {
	if f.IsTest() {
		return kinds.Test
	}
	if f.IsCmd() {
		return kinds.Bin
	}
	return kinds.Lib
}

// IsGenerated returns whether a file is generated this is copied from
// https://go-review.googlesource.com/c/go/+/487935/8/src/go/ast/ast.go
func IsGenerated(file *ast.File) bool {
	_, ok := generator(file)
	return ok
}

func generator(file *ast.File) (string, bool) {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if comment.Pos() > file.Package {
				break // after package declaration
			}
			// opt: check Contains first to avoid unnecessary array allocation in Split.
			const prefix = "// Code generated "
			if strings.Contains(comment.Text, prefix) {
				for _, line := range strings.Split(comment.Text, "\n") {
					if rest, ok := strings.CutPrefix(line, prefix); ok {
						if gen, ok := strings.CutSuffix(rest, " DO NOT EDIT."); ok {
							return gen, true
						}
					}
				}
			}
		}
	}
	return "", false
}
