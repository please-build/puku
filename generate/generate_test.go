package generate

import (
	"testing"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/kinds"
	"github.com/please-build/puku/trie"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestAllocateSources(t *testing.T) {
	foo := newRule(newRuleExpr("go_library", "foo"), kinds.DefaultKinds["go_library"], "")
	fooTest := newRule(newRuleExpr("go_test", "foo_test"), kinds.DefaultKinds["go_test"], "")

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

func TestAllocateSourcesToCustomKind(t *testing.T) {
	exampleKind := &kinds.Kind{
		Name: "go_example_lib",
		Type: kinds.Lib,
	}

	satKind := &kinds.Kind{
		Name: "service_acceptance_test",
		Type: kinds.Test,
	}

	foo := newRule(newRuleExpr("go_example_lib", "foo"), exampleKind, "")
	fooTest := newRule(newRuleExpr("go_test", "foo_test"), satKind, "")

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

	assert.Len(t, newRules, 0)

	assert.Equal(t, []string{"foo.go", "bar.go"}, mustGetSources(t, rules[0]))
	assert.Equal(t, []string{"foo_test.go", "bar_test.go"}, mustGetSources(t, rules[1]))
}

func TestUpdateDeps(t *testing.T) {
	type ruleKind struct {
		kind *kinds.Kind
		srcs []string
	}

	testCases := []struct {
		name         string
		srcs         []*GoFile
		rules        map[string]*ruleKind
		expectedDeps map[string][]string
		modules      []string
		installs     map[string]string
		conf         *config.Config
		proxy        FakeProxy
	}{
		{
			name: "adds import from known module",
			srcs: []*GoFile{
				{
					FileName: "foo.go",
					Imports:  []string{"github.com/example/module/foo"},
					Name:     "foo",
				},
			},
			modules: []string{"github.com/example/module"},
			rules: map[string]*ruleKind{
				"foo": {
					srcs: []string{"foo.go"},
					kind: kinds.DefaultKinds["go_library"],
				},
			},
			expectedDeps: map[string][]string{
				"foo": {"///third_party/go/github.com_example_module//foo:foo"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := &Update{
				modules:      tc.modules,
				importPath:   "github.com/this/module",
				installs:     trie.New(),
				knownImports: map[string]string{},
				proxy:        tc.proxy,
			}

			for path, value := range tc.installs {
				u.installs.Add(path, value)
			}

			conf := tc.conf
			if conf == nil {
				conf = new(config.Config)
			}

			rules := make([]*rule, 0, len(tc.rules))
			ruleMap := make(map[string]*rule, len(tc.rules))
			for name, r := range tc.rules {
				newRule := newRule(newRuleExpr(r.kind.Name, name), r.kind, "")
				for _, src := range r.srcs {
					newRule.addSrc(src)
				}
				rules = append(rules, newRule)
				ruleMap[name] = newRule
			}

			files := make(map[string]*GoFile, len(tc.srcs))
			for _, f := range tc.srcs {
				files[f.FileName] = f
			}

			for name, deps := range tc.expectedDeps {
				rule := ruleMap[name]
				err := u.updateDeps(conf, rule, rules, files)
				require.NoError(t, err)
				assert.Equal(t, deps, rule.AttrStrings("deps"))
			}
		})
	}

}

func mustGetSources(t *testing.T, rule *rule) []string {
	t.Helper()

	srcs, err := rule.allSources()
	require.NoError(t, err)
	return srcs
}
