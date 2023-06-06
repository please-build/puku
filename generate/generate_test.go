package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdate(t *testing.T) {
	rules := []*Rule{
		{
			name: "foo",
			srcs: []string{"foo.go"},
		},
		{
			name: "foo_test",
			srcs: []string{"foo_test.go"},
			test: true,
		},
	}

	files := map[string]*GoFile{
		"bar.go": {
			Name: "foo",
		},
		"bar_test.go": {
			Name: "foo",
			Test: true,
		},
		"external_test.go": {
			Name: "foo_test",
			Test: true,
		},
		"foo.go": {
			Name: "foo",
		},
		"foo_test.go": {
			Name: "foo",
			Test: true,
		},
	}

	unallocated := []string{"bar.go", "bar_test.go", "external_test.go"}

	newRules, err := allocateSources("foo", files, unallocated, rules)
	if err != nil {
		panic(err)
	}

	assert.Len(t, newRules, 1)
	assert.Equal(t, newRules[0].name, "foo_test")
	assert.Equal(t, newRules[0].srcs, []string{"external_test.go"})

	assert.Equal(t, rules[0].srcs, []string{"foo.go", "bar.go"})
	assert.Equal(t, rules[1].srcs, []string{"foo_test.go", "bar_test.go"})
}
