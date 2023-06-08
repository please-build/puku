package generate

import (
	"github.com/please-build/puku/trie"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllocateSources(t *testing.T) {
	foo := newRule(newRuleExpr("go_library", "foo"), KindTypeLib, "")
	fooTest := newRule(newRuleExpr("go_test", "foo_test"), KindTypeTest, "")

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

func TestUpdateDeps(t *testing.T) {
	foo := newRule(newRuleExpr("go_library", "foo"), KindTypeLib, "")
	foo.addSrc("foo.go")
	foo.addSrc("bar.go")

	fooTest := newRule(newRuleExpr("go_test", "foo"), KindTypeTest, "")
	fooTest.addSrc("foo_test.go")

	u := &Update{
		modules:       []string{"github.com/example/module"},
		importPath:    "github.com/this/module",
		thirdPartyDir: "third_party/go",
		installs:      trie.New(),
		knownImports:  map[string]string{},
	}

	files := map[string]*GoFile{
		"foo.go": {
			Name:     "foo",
			FileName: "foo.go",
			Imports:  []string{"github.com/example/module/pkg", "io"},
		},
		"bar.go": {
			Name:     "foo",
			FileName: "bar.go",
			Imports:  []string{"github.com/example/module", "io"},
		},
		"foo_test.go": {
			Name:     "foo",
			FileName: "foo_test.go",
			Imports:  []string{},
		},
	}

	rules := []*rule{foo, fooTest}

	err := u.updateDeps(foo, rules, files)
	require.NoError(t, err)

	deps := foo.AttrStrings("deps")
	require.Len(t, deps, 2)
	assert.Contains(t, deps, "///third_party/go/github.com_example_module//pkg:pkg")
	assert.Contains(t, deps, "///third_party/go/github.com_example_module//:module")

	err = u.updateDeps(fooTest, rules, files)
	require.NoError(t, err)

	deps = fooTest.AttrStrings("deps")
	require.Len(t, deps, 1)

	assert.Contains(t, deps, ":foo")
}

func mustGetSources(t *testing.T, rule *rule) []string {
	t.Helper()

	srcs, err := rule.allSources()
	require.NoError(t, err)
	return srcs
}
