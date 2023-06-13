package main

import (
	"github.com/please-build/puku/config"
	"github.com/please-build/puku/please"
	"os"

	"github.com/peterebden/go-cli-init/v5/flags"
	"github.com/peterebden/go-cli-init/v5/logging"

	"github.com/please-build/puku/generate"
	"github.com/please-build/puku/watch"
	"github.com/please-build/puku/work"
)

var opts = struct {
	Usage    string
	LintOnly bool `long:"nowrite" description:"Prints corrections to stdout instead of formatting the files"`
	Watch    bool `long:"watch" description:"Watch the directory"`
	Args     struct {
		Paths []string `positional-arg-name:"packages" description:"The packages to process"`
	} `positional-args:"true"`
}{
	Usage: `
puku is a tool used to generate and update Go targets in build files
`,
}

var log = logging.MustGetLogger()

func main() {
	root, err := work.FindRoot()
	if err != nil {
		log.Fatalf("%v", err)
	}

	if err := os.Chdir(root); err != nil {
		log.Fatalf("failed to set working dir to repo root: %v", err)
	}

	flags.ParseFlagsOrDie("puku", &opts, nil)

	if opts.LintOnly && opts.Watch {
		log.Fatalf("watch mode doesn't support --nowrite")
	}

	conf, err := config.ReadConfig(".")
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	plzConf, err := please.QueryConfig(conf.GetPlzPath())
	if err != nil {
		log.Fatalf("failed to query config: %w", err)
	}

	u := generate.NewUpdate(!opts.LintOnly, plzConf)

	paths := opts.Args.Paths
	if len(opts.Args.Paths) == 0 {
		paths = []string{"..."}
	}
	if opts.Watch {
		err := watch.Watch(u, paths...)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	if err := u.Update(paths...); err != nil {
		log.Fatalf("%v", err)
	}
}
