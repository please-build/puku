package main

import (
	"github.com/peterebden/go-cli-init/v5/flags"
	"github.com/peterebden/go-cli-init/v5/logging"

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
	flags.ParseFlagsOrDie("puku", &opts, nil)
	g := generate.NewUpdate("plz", "third_party/go", []string{"BUILD", "BUILD.plz"})
	if err := g.Update(opts.Args.Paths); err != nil {
		log.Fatalf("%v", err)
	}

}
