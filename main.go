package main

import (
	"os"

	"github.com/peterebden/go-cli-init/v5/flags"
	"github.com/peterebden/go-cli-init/v5/logging"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/generate"
)

var opts = struct {
	Usage string
	Args  struct {
		Paths []string `positional-arg-name:"packages" description:"The packages to process"`
	} `positional-args:"true"`
}{
	Usage: `
puku is a tool used to generate and update Go targets in build files
`,
}

var log = logging.MustGetLogger()

func main() {
	root, err := config.FindRepoRoot()
	if err != nil {
		log.Fatalf("%v", err)
	}

	if err := os.Chdir(root); err != nil {
		log.Fatalf("failed to set working dir to repo root: %v", err)
	}
	flags.ParseFlagsOrDie("puku", &opts, nil)
	g := generate.NewUpdate("plz", "third_party/go")

	paths := opts.Args.Paths
	if len(opts.Args.Paths) == 0 {
		paths = []string{"..."}
	}

	if err := g.Update(paths); err != nil {
		log.Fatalf("%v", err)
	}
}
