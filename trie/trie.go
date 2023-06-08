package trie

import (
	"strings"
)

func New() *Trie {
	return &Trie{children: map[string]*Trie{}}
}

type Trie struct {
	children map[string]*Trie
	matchAll bool
	value    string
}

func (trie *Trie) Add(path, value string) {
	trie.add(strings.Split(path, "/"), value)
}

func (trie *Trie) add(parts []string, value string) {
	if len(parts) == 0 {
		trie.value = value
		return
	}

	key := parts[0]
	if key == "..." {
		trie.matchAll = true
		trie.children = nil
		trie.value = value
		return
	}

	var next *Trie
	if n, ok := trie.children[key]; ok {
		next = n
	} else {
		next = &Trie{children: map[string]*Trie{}}
		trie.children[key] = next
	}
	next.add(parts[1:], value)
}

func (trie *Trie) Get(path string) string {
	return trie.get(strings.Split(path, "/"))
}

func (trie *Trie) get(parts []string) string {
	if trie.matchAll {
		return trie.value
	}

	if len(parts) == 0 {
		return trie.value
	}

	next, ok := trie.children[parts[0]]
	if !ok {
		return ""
	}

	return next.get(parts[1:])
}
