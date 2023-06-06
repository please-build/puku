package generate

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type GoFile struct {
	// Name is the name from the package clause of this file
	Name string
	// Imports are the imports of this file
	Imports []string
	// Cgo is whether this file import Cgo
	Cgo, Test, Cmd bool
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
	isCgo := false
	imports := make([]string, 0, len(f.Imports))
	for _, i := range f.Imports {
		path := i.Path.Value

		path = path[1 : len(path)-1] // remove quotes
		if path == "C" {
			isCgo = true
		}
		imports = append(imports, path)
	}
	return &GoFile{
		Name:    f.Name.Name,
		Imports: imports,
		Cgo:     isCgo,
		Test:    strings.HasSuffix(src, "_test.go"),
		Cmd:     f.Name.Name == "main",
	}, nil
}
