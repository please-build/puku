package generate

import (
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllocateSources(t *testing.T) {
	foo := newRule(newRuleExpr("go_library", "foo"), KindType_Lib, "")
	fooTest := newRule(newRuleExpr("go_test", "foo_test"), KindType_Test, "")

	foo.addSrc("foo.go")
	fooTest.addSrc("foo_test.go")

	rules := []*rule{foo, fooTest}

	files := map[string]*GoFile{
		"bar.go": {
			Name:     "foo",
			FileName: "bar.go",
		},
		"bar_test.go": {
			Name:     "foo",
			FileName: "bar_test.go",
		},
		"external_test.go": {
			Name:     "foo_test",
			FileName: "external_test.go",
		},
		"foo.go": {
			Name:     "foo",
			FileName: "foo.go",
		},
		"foo_test.go": {
			Name:     "foo",
			FileName: "foo_test.go",
		},
	}

	u := new(Update)
	newRules, err := u.allocateSources("foo", files, rules)
	if err != nil {
		panic(err)
	}

	assert.Len(t, newRules, 1)
	assert.Equal(t, "foo_test", newRules[0].Name())
	assert.Equal(t, []string{"external_test.go"}, mustGetSources(t, newRules[0]))

	assert.Equal(t, []string{"foo.go", "bar.go"}, mustGetSources(t, rules[0]))
	assert.Equal(t, []string{"foo_test.go", "bar_test.go"}, mustGetSources(t, rules[1]))
}

func mustGetSources(t *testing.T, rule *rule) []string {
	t.Helper()

	srcs, err := rule.allSources()
	require.NoError(t, err)
	return srcs
}
