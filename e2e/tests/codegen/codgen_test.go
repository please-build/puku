package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/please-build/puku/e2e/harness"
	"github.com/please-build/puku/edit"
)

func TestTarget(t *testing.T) {
	h := harness.MustNew()
	err := h.Format("//codegen/...")
	require.NoError(t, err)

	file, err := h.ParseFile("codegen/BUILD_FILE.plz")
	require.NoError(t, err)

	codegen := edit.FindTargetByName(file, "codegen")
	require.NotNil(t, codegen)

	assert.ElementsMatch(t, []string{"//foo"}, codegen.AttrStrings("deps"))
	assert.ElementsMatch(t, []string{":srcs"}, codegen.AttrStrings("srcs"))

	codegenProvider := edit.FindTargetByName(file, "codegen_provider")
	require.NotNil(t, codegenProvider)

	assert.ElementsMatch(t, []string{"//foo"}, codegenProvider.AttrStrings("deps"))
	assert.ElementsMatch(t, []string{":provider"}, codegenProvider.AttrStrings("srcs"))

	recursiveCodegenProvider := edit.FindTargetByName(file, "recursive_codegen_provider")
	require.NotNil(t, recursiveCodegenProvider)

	assert.ElementsMatch(t, []string{"//foo"}, recursiveCodegenProvider.AttrStrings("deps"))
	assert.ElementsMatch(t, []string{":recursive_provider"}, recursiveCodegenProvider.AttrStrings("srcs"))
}
