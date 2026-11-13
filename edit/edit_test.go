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

// TestRemoveTarget checks that exactly the requested rule is removed, even when the file contains statements that
// aren't rules (such as comment blocks and assignments) ahead of it. Previously the rule's index among the file's rules
// was used as an index into file.Stmt, which removed the wrong statement in these cases.
func TestRemoveTarget(t *testing.T) {
	for _, tc := range []struct {
		name      string
		src       string
		remove    string
		wantRules []string
	}{
		{
			name:      "only rules",
			src:       "go_repo(name = \"a\")\ngo_repo(name = \"b\")\ngo_repo(name = \"c\")\n",
			remove:    "b",
			wantRules: []string{"a", "c"},
		},
		{
			name:      "standalone comment block before the rules",
			src:       "# header\n\ngo_repo(name = \"a\")\ngo_repo(name = \"b\")\ngo_repo(name = \"c\")\n",
			remove:    "b",
			wantRules: []string{"a", "c"},
		},
		{
			name:      "assignment and comment blocks before the rules",
			src:       "# header\n\nVERSION = \"1\"\n\ngo_repo(name = \"a\")\n\n# comment\n\ngo_repo(name = \"b\")\ngo_repo(name = \"c\")\n",
			remove:    "c",
			wantRules: []string{"a", "b"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := build.Parse("BUILD", []byte(tc.src))
			require.NoError(t, err)

			assert.True(t, RemoveTarget(file, FindTargetByName(file, tc.remove)))

			rules := file.Rules("")
			got := make([]string, 0, len(rules))
			for _, r := range rules {
				got = append(got, r.Name())
			}
			assert.Equal(t, tc.wantRules, got)
		})
	}
}
