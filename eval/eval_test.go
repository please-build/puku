package eval

import (
	"testing"

	"github.com/bazelbuild/buildtools/build"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
