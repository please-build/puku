package trie

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTrie(t *testing.T) {
	installs := map[string]string{
		"//third_party/go:bar": "github.com/foo/bar/...",
		"//third_party/go:baz": "github.com/foo/baz",
		"//third_party/go:v1":  "github.com/some/module/...",
		"//third_party/go:v2":  "github.com/some/module/v2/...",
	}

	trie := New()

	for k, install := range installs {
		trie.Add(install, k)
	}

	assert.Equal(t, "//third_party/go:bar", trie.Get("github.com/foo/bar"))
	assert.Equal(t, "//third_party/go:bar", trie.Get("github.com/foo/bar/baz"))
	assert.Equal(t, "//third_party/go:baz", trie.Get("github.com/foo/baz"))
	assert.Equal(t, "//third_party/go:v1", trie.Get("github.com/some/module/foo"))
	assert.Equal(t, "//third_party/go:v2", trie.Get("github.com/some/module/v2/bar"))
	assert.Equal(t, "", trie.Get("github.com/foo/baz/bar"))
}
