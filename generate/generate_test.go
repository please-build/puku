package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllocateSources(t *testing.T) {
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
	assert.Equal(t, "foo_test", newRules[0].name)
	assert.Equal(t, []string{"external_test.go"}, newRules[0].srcs)

	assert.Equal(t, []string{"foo.go", "bar.go"}, rules[0].srcs)
	assert.Equal(t, []string{"foo_test.go", "bar_test.go"}, rules[1].srcs)
}
