package generate

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/please-build/puku/kinds"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

type fileType string

const (
	GO fileType = "GO"
	TS          = "TS"
)

// SourceFile represents a single source file in the repo.
type SourceFile struct {
	name, fileName string
	imports        []string
	dir            string
	fileType       fileType
	// // Name is the name from the package clause of this file
	// Name() string
	// // FileName is the name of the file
	// FileName() string
	// // Dir is the directory of the file
	// Dir() string
	// // Imports are the imports of this file
	// Imports() []string

	// IsExternal(pkgName string) bool
	// IsTest() bool
	// IsCmd() bool

	// KindType() kinds.Type
}

func (f *SourceFile) Name() string {
	if f.fileType == TS {
		// Remove extension from file name
		return strings.TrimSuffix(f.name, filepath.Ext(f.name))
	}
	return f.name
}

func (f *SourceFile) FileName() string {
	return f.fileName
}

func (f *SourceFile) Dir() string {
	return f.dir
}

func (f *SourceFile) Imports() []string {
	return f.imports
}

// IsExternal returns whether the test is external
func (f *SourceFile) IsExternal(pkgName string) bool {
	if f.fileType == TS {
		return false
	}
	return f.name == filepath.Base(pkgName)+"_test" && f.IsTest()
}

func (f *SourceFile) IsTest() bool {
	if f.fileType == TS {
		return strings.Contains(f.FileName(), ".spec.")
	}
	return strings.HasSuffix(f.fileName, "_test.go")
}

func (f *SourceFile) IsCmd() bool {
	if f.fileType == TS {
		return false
	}
	return f.name == "main"
}

func (f *SourceFile) KindType() kinds.Type {
	if f.IsTest() {
		return kinds.Test
	}
	if f.IsCmd() {
		return kinds.Bin
	}
	return kinds.Lib
}

// ImportDir does _some_ of what the go/build ImportDir does but is more permissive.
func ImportDir(dir string) (map[string]*SourceFile, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*SourceFile, len(files))
	for _, info := range files {
		if !info.Type().IsRegular() {
			continue
		}

		fileExtension := filepath.Ext(info.Name())

		if fileExtension == ".go" {
			f, err := importGoFile(dir, info.Name())
			if err != nil {
				return nil, err
			}
			ret[info.Name()] = f
		}

		if fileExtension == ".ts" || fileExtension == ".tsx" {
			f, err := importTsFile(dir, info.Name())
			if err != nil {
				return nil, err
			}
			ret[info.Name()] = f
		}
	}

	return ret, nil
}

func importGoFile(dir, src string) (*SourceFile, error) {
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, src), nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return nil, err
	}
	imports := make([]string, 0, len(f.Imports))
	for _, i := range f.Imports {
		path := i.Path.Value
		path = strings.Trim(path, `"`)
		imports = append(imports, path)
	}

	return &SourceFile{
		name:     f.Name.Name,
		fileName: src,
		imports:  imports,
		dir:      dir,
		fileType: GO,
	}, nil
}

func importTsFile(dir, src string) (*SourceFile, error) {
	tsParser := sitter.NewParser()
	tsParser.SetLanguage(typescript.GetLanguage())

	sourceCode, err := os.ReadFile(filepath.Join(dir, src))
	if err != nil {
		return nil, err
	}

	log.Debugf("Parsing TS file: %s\n", filepath.Join(dir, src))

	ctx := context.TODO()
	tree, err := tsParser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, err
	}

	var imports []string

	n := tree.RootNode()
	cursor := sitter.NewTreeCursor(n)
	defer cursor.Close()

	// enter tree
	cursor.GoToFirstChild()

	for true {
		node := cursor.CurrentNode()
		nodeType := node.Type()

		// we only care about import statements
		if nodeType == "import_statement" {
			importCursor := sitter.NewTreeCursor(node)
			defer importCursor.Close()
			importCursor.GoToFirstChild()

			for true {
				if importCursor.CurrentFieldName() == "source" {
					// remove quotes around string
					importPath := string(sourceCode[importCursor.CurrentNode().StartByte()+1 : importCursor.CurrentNode().EndByte()-1])
					imports = append(imports, importPath)
				}

				result := importCursor.GoToNextSibling()
				if !result {
					break
				}
			}
		}

		if nodeType == "export_statement" {
			exportCursor := sitter.NewTreeCursor(node)
			defer exportCursor.Close()
			exportCursor.GoToFirstChild()

			for true {
				if exportCursor.CurrentNode().Type() == "from" {
					// Go to the next sibling to get from path
					exportCursor.GoToNextSibling()
					// remove quotes around string
					importPath := string(sourceCode[exportCursor.CurrentNode().StartByte()+1 : exportCursor.CurrentNode().EndByte()-1])
					imports = append(imports, importPath)
				}

				result := exportCursor.GoToNextSibling()
				if !result {
					break
				}
			}
		}

		result := cursor.GoToNextSibling()
		if !result {
			break
		}
	}

	return &SourceFile{
		name:     src,
		fileName: src,
		imports:  imports,
		dir:      dir,
		fileType: TS,
	}, nil
}
