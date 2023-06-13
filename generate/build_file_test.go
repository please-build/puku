package generate

import (
	"testing"

	"github.com/bazelbuild/buildtools/build"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var buildFileNames = []string{"BUILD_FILE", "BUILD_FILE.plz"}

func TestParseBuildFile(t *testing.T) {
	f, err := parseBuildFile("generate/test_data", buildFileNames)
	require.NoError(t, err)

	libs := f.Rules("go_library")
	require.Len(t, libs, 1)

	f, err = parseBuildFile("generate/test_data/foo", buildFileNames)
	require.NoError(t, err)

	libs = f.Rules("go_library")
	require.Len(t, libs, 1)

	f, err = parseBuildFile("generate/test_data/foo/bar", buildFileNames)
	require.NoError(t, err)
	assert.Equal(t, "generate/test_data/foo/bar/BUILD_FILE", f.Path)
}

func TestEnsureSubinclude(t *testing.T) {
	t.Run("adds if missing", func(t *testing.T) {
		file, _ := build.Parse("test", nil)
		ensureSubinclude(file)
		require.Len(t, file.Stmt, 1)
		subinc := &build.CallExpr{
			X:    &build.Ident{Name: "subinclude"},
			List: []build.Expr{newStringExpr("///go//build_defs:go")},
		}
		assert.Equal(t, file.Stmt[0], subinc)
	})

	t.Run("updates existing", func(t *testing.T) {
		file, _ := build.Parse("test", nil)
		file.Stmt = []build.Expr{
			&build.CallExpr{
				X:    &build.Ident{Name: "subinclude"},
				List: []build.Expr{newStringExpr("///python//build_defs:python")},
			},
		}

		ensureSubinclude(file)
		require.Len(t, file.Stmt, 1)
		subinc := &build.CallExpr{
			X: &build.Ident{Name: "subinclude"},
			List: []build.Expr{
				newStringExpr("///python//build_defs:python"),
				newStringExpr("///go//build_defs:go"),
			},
		}
		assert.Equal(t, file.Stmt[0], subinc)
	})

	t.Run("does nothing if it exists", func(t *testing.T) {
		file, _ := build.Parse("test", nil)
		file.Stmt = []build.Expr{
			&build.CallExpr{
				X:    &build.Ident{Name: "subinclude"},
				List: []build.Expr{newStringExpr("///go//build_defs:go")},
			},
		}

		ensureSubinclude(file)
		require.Len(t, file.Stmt, 1)
		subinc := &build.CallExpr{
			X: &build.Ident{Name: "subinclude"},
			List: []build.Expr{
				newStringExpr("///go//build_defs:go"),
			},
		}
		assert.Equal(t, file.Stmt[0], subinc)
	})
}
