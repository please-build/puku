package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/kinds"
	"github.com/please-build/puku/options"
	"github.com/please-build/puku/please"
)

func TestAllocateSources(t *testing.T) {
	foo := edit.NewRule(edit.NewRuleExpr("go_library", "foo"), kinds.DefaultKinds["go_library"], "")
	fooTest := edit.NewRule(edit.NewRuleExpr("go_test", "foo_test"), kinds.DefaultKinds["go_test"], "")

	foo.AddSrc("foo.go")
	fooTest.AddSrc("foo_test.go")

	rules := []*edit.Rule{foo, fooTest}

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

	u := newUpdater(new(please.Config), options.TestOptions)
	conf := &config.Config{PleasePath: "plz"}
	newRules, err := u.allocateSources(conf, "foo", files, rules)
	require.NoError(t, err)

	require.Len(t, newRules, 1)
	assert.Equal(t, "foo_test", newRules[0].Name())
	assert.ElementsMatch(t, []string{"external_test.go"}, mustGetSources(t, u, newRules[0]))

	assert.ElementsMatch(t, []string{"foo.go", "bar.go"}, mustGetSources(t, u, rules[0]))
	assert.ElementsMatch(t, []string{"foo_test.go", "bar_test.go"}, mustGetSources(t, u, rules[1]))
}

func TestAddingLibDepToTest(t *testing.T) {
	foo := edit.NewRule(edit.NewRuleExpr("go_library", "foo"), kinds.DefaultKinds["go_library"], "")
	fooTest := edit.NewRule(edit.NewRuleExpr("go_test", "foo_test"), kinds.DefaultKinds["go_test"], "")

	files := map[string]*GoFile{
		"foo.go": {
			Name:     "foo",
			FileName: "foo.go",
		},
		"foo_test.go": {
			Name:     "foo",
			FileName: "foo_test.go",
		},
	}

	foo.SetAttr(foo.SrcsAttr(), edit.NewStringList([]string{"foo.go"}))
	fooTest.SetAttr(fooTest.SrcsAttr(), edit.NewStringList([]string{"foo_test.go"}))

	u := newUpdater(new(please.Config), options.TestOptions)
	conf := &config.Config{PleasePath: "plz"}
	err := u.updateRuleDeps(conf, fooTest, []*edit.Rule{foo, fooTest}, files)
	require.NoError(t, err)

	assert.Equal(t, fooTest.AttrStrings("deps"), []string{":foo"})
}

func TestAllocateSourcesToCustomKind(t *testing.T) {
	exampleKind := &kinds.Kind{
		Name:     "go_example_lib",
		Type:     kinds.Lib,
		SrcsAttr: "go_srcs",
	}

	satKind := &kinds.Kind{
		Name: "service_acceptance_test",
		Type: kinds.Test,
	}

	foo := edit.NewRule(edit.NewRuleExpr("go_example_lib", "foo"), exampleKind, "")
	fooTest := edit.NewRule(edit.NewRuleExpr("go_test", "foo_test"), satKind, "")

	foo.AddSrc("foo.go")
	fooTest.AddSrc("foo_test.go")

	rules := []*edit.Rule{foo, fooTest}

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

	u := newUpdater(new(please.Config), options.TestOptions)
	conf := &config.Config{PleasePath: "plz"}
	newRules, err := u.allocateSources(conf, "foo", files, rules)
	require.NoError(t, err)

	assert.Len(t, newRules, 0)

	assert.ElementsMatch(t, []string{"foo.go", "bar.go"}, mustGetSources(t, u, rules[0]))
	assert.Equal(t, rules[0].SrcsAttr(), exampleKind.SrcsAttr)
	assert.ElementsMatch(t, []string{"foo_test.go", "bar_test.go"}, mustGetSources(t, u, rules[1]))
}

func TestAllocateSourcesToNonGoKind(t *testing.T) {
	exampleKind := &kinds.Kind{
		Name:         "go_example_lib",
		Type:         kinds.Lib,
		NonGoSources: true,
	}

	foo := edit.NewRule(edit.NewRuleExpr("go_example_lib", "nogo"), exampleKind, "")

	rules := []*edit.Rule{foo}

	files := map[string]*GoFile{
		"foo.go": {
			Name:     "foo",
			FileName: "foo.go",
		},
	}

	u := newUpdater(new(please.Config), options.TestOptions)
	u.plzConf = &please.Config{}
	newRules, err := u.allocateSources(new(config.Config), "foo", files, rules)
	require.NoError(t, err)

	require.Len(t, newRules, 1)

	assert.ElementsMatch(t, []string{}, mustGetSources(t, u, foo))
	assert.Equal(t, "go_library", newRules[0].Rule.Kind())
	assert.ElementsMatch(t, []string{"foo.go"}, mustGetSources(t, u, newRules[0]))
}

func TestUpdateDeps(t *testing.T) {
	type ruleKind struct {
		kind *kinds.Kind
		srcs []string
	}

	testCases := []struct {
		name         string
		srcs         []*GoFile
		rule         *ruleKind
		expectedDeps []string
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
			rule: &ruleKind{
				srcs: []string{"foo.go"},
				kind: kinds.DefaultKinds["go_library"],
			},
			expectedDeps: []string{"///third_party/go/github.com_example_module//foo"},
		},
		{
			name: "handles installs",
			srcs: []*GoFile{
				{
					FileName: "foo.go",
					Imports: []string{
						"github.com/example/module1/foo",
						"github.com/example/module2/foo/bar/baz",
					},
					Name: "foo",
				},
			},
			modules: []string{},
			installs: map[string]string{
				"github.com/example/module1/foo": "//third_party/go:module1",
				"github.com/example/module2/...": "//third_party/go:module2",
			},
			rule: &ruleKind{
				srcs: []string{"foo.go"},
				kind: kinds.DefaultKinds["go_library"],
			},

			expectedDeps: []string{"//third_party/go:module1", "//third_party/go:module2"},
		},
		{
			name: "handles custom kinds",
			srcs: []*GoFile{
				{
					FileName: "foo.go",
					Imports: []string{
						"github.com/example/module/foo",
						"github.com/example/module/bar",
					},
					Name: "foo",
				},
			},
			modules: []string{"github.com/example/module"},
			rule: &ruleKind{
				srcs: []string{"foo.go"},
				kind: &kinds.Kind{
					Name:         "example_library",
					Type:         kinds.Lib,
					ProvidedDeps: []string{"///third_party/go/github.com_example_module//foo"},
				},
			},
			expectedDeps: []string{"///third_party/go/github.com_example_module//bar"},
		},
		{
			name:    "handles missing src",
			srcs:    []*GoFile{},
			modules: []string{"github.com/example/module"},
			rule: &ruleKind{
				srcs: []string{"foo.go"},
				kind: &kinds.Kind{
					Name:         "example_library",
					Type:         kinds.Lib,
					ProvidedDeps: []string{"///third_party/go/github.com_example_module//foo"},
				},
			},
			expectedDeps: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plzConf := new(please.Config)
			plzConf.Plugin.Go.ImportPath = []string{"github.com/this/module"}
			u := newUpdater(plzConf, options.TestOptions)
			u.modules = tc.modules
			u.proxy = tc.proxy

			for path, value := range tc.installs {
				u.installs.Add(path, value)
			}

			conf := tc.conf
			if conf == nil {
				conf = new(config.Config)
			}

			r := edit.NewRule(edit.NewRuleExpr(tc.rule.kind.Name, "rule"), tc.rule.kind, "")
			for _, src := range tc.rule.srcs {
				r.AddSrc(src)
			}

			files := make(map[string]*GoFile, len(tc.srcs))
			srcNames := make([]string, 0, len(tc.srcs))
			for _, f := range tc.srcs {
				files[f.FileName] = f
				srcNames = append(srcNames, f.FileName)
			}

			err := u.updateRuleDeps(conf, r, []*edit.Rule{}, files)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.expectedDeps, r.AttrStrings("deps"))
			assert.ElementsMatch(t, srcNames, r.AttrStrings(r.SrcsAttr()))
		})
	}
}

func mustGetSources(t *testing.T, u *updater, rule *edit.Rule) []string {
	t.Helper()

	srcs, err := u.eval.EvalGlobs(rule.Dir, rule.Rule, rule.SrcsAttr())
	require.NoError(t, err)
	return srcs
}
