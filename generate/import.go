package generate

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
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

		f, err := ImportFile(dir, info.Name())
		if err != nil {
			return nil, err
		}

		ret[info.Name()] = f
	}

	return ret, nil
}

func ImportFile(dir, src string) (*GoFile, error) {
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, src), nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	imports := make([]string, 0, len(f.Imports))
	for _, i := range f.Imports {
		path := i.Path.Value
		path = path[1 : len(path)-1] // remove quotes
		imports = append(imports, path)
	}
	return &GoFile{
		Name:     f.Name.Name,
		FileName: src,
		Imports:  imports,
	}, nil
}

func (f *GoFile) IsCgo() bool {
	for _, i := range f.Imports {
		if i == "C" {
			return true
		}
	}
	return false
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
