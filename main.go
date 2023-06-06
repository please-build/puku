package main

import (
	"os"

	"github.com/please-build/paku/generate"
)

func main() {
	g := generate.NewUpdate("plz", "third_party/go", []string{"BUILD", "BUILD.plz"})
	if err := g.Update(os.Args[1:]); err != nil {
		panic(err)
	}
}
