package edit

import (
	"testing"

	"github.com/please-build/buildtools/build"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureSubinclude(t *testing.T) {
	t.Run("adds if missing", func(t *testing.T) {
		file, _ := build.Parse("test", nil)

		EnsureSubinclude(file)
		require.Len(t, file.Stmt, 1)
		subinc := &build.CallExpr{
			X:    &build.Ident{Name: "subinclude"},
			List: []build.Expr{NewStringExpr("///go//build_defs:go")},
		}
		assert.Equal(t, file.Stmt[0], subinc)
	})

	t.Run("updates existing", func(t *testing.T) {
		file, _ := build.Parse("test", nil)
		file.Stmt = []build.Expr{
			&build.CallExpr{
				X:    &build.Ident{Name: "subinclude"},
				List: []build.Expr{NewStringExpr("///python//build_defs:python")},
			},
		}

		EnsureSubinclude(file)
		require.Len(t, file.Stmt, 1)
		subinc := &build.CallExpr{
			X: &build.Ident{Name: "subinclude"},
			List: []build.Expr{
				NewStringExpr("///python//build_defs:python"),
				NewStringExpr("///go//build_defs:go"),
			},
		}
		assert.Equal(t, file.Stmt[0], subinc)
	})

	t.Run("does nothing if it exists", func(t *testing.T) {
		file, _ := build.Parse("test", nil)
		file.Stmt = []build.Expr{
			&build.CallExpr{
				X:    &build.Ident{Name: "subinclude"},
				List: []build.Expr{NewStringExpr("///go//build_defs:go")},
			},
		}

		EnsureSubinclude(file)
		require.Len(t, file.Stmt, 1)
		subinc := &build.CallExpr{
			X: &build.Ident{Name: "subinclude"},
			List: []build.Expr{
				NewStringExpr("///go//build_defs:go"),
			},
		}
		assert.Equal(t, file.Stmt[0], subinc)
	})
}
