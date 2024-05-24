package eval

import (
	"testing"

	"github.com/please-build/buildtools/build"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/please-build/puku/glob"
)

func TestParseGlob(t *testing.T) {
	testCases := []struct {
		name    string
		code    string
		include []string
		exclude []string
	}{
		{
			name:    "both positional",
			code:    `glob(["*.go"], ["*_test.go"])`,
			include: []string{"*.go"},
			exclude: []string{"*_test.go"},
		},
		{
			name:    "mixed positional and named",
			code:    `glob(["*.go"], exclude = ["*_test.go"])`,
			include: []string{"*.go"},
			exclude: []string{"*_test.go"},
		},
		{
			name:    "both named",
			code:    `glob(include = ["*.go"], exclude = ["*_test.go"])`,
			include: []string{"*.go"},
			exclude: []string{"*_test.go"},
		},
		{
			name: "ignores other args",
			// Please has some other stuff we don't care about.
			code:    `glob(include = ["*.go"], foo = True, bar = True)`,
			include: []string{"*.go"},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			file, err := build.ParseBuild(test.name, []byte(test.code))
			require.NoError(t, err)
			require.Len(t, file.Stmt, 1)
			call, ok := file.Stmt[0].(*build.CallExpr)
			require.True(t, ok)

			args := parseGlob(call)
			assert.Equal(t, test.include, args.Include)
			assert.Equal(t, test.exclude, args.Exclude)
		})
	}
}
func TestEvalGlob(t *testing.T) {
	e := New(glob.New())
	testCases := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "glob + glob",
			code:     `glob(["mai*.go"]) + glob(["ba*.go"])`,
			expected: []string{"main.go", "bar.go", "bar_test.go"},
		},
		{
			name:     "glob + glob + strings",
			code:     `glob(["mai*.go"]) + glob(["*_test.go"]) + ["bar.go"]`,
			expected: []string{"main.go", "bar.go", "bar_test.go"},
		},
		{
			name:     "strings + strings",
			code:     `["main.go"] + ["bar.go"]`,
			expected: []string{"main.go", "bar.go"},
		},
		{
			name:     "glob + strings",
			code:     `glob(["mai*.go"]) + ["bar.go"]`,
			expected: []string{"main.go", "bar.go"},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			file, err := build.ParseBuild(test.name, []byte(test.code))
			require.NoError(t, err)
			require.Len(t, file.Stmt, 1)
			got, err := e.EvalGlobs("test_project", file.Stmt[0])
			require.NoError(t, err)
			assert.ElementsMatch(t, test.expected, got)
		})
	}
}
