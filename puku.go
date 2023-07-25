package main

import (
	"os"
	"path/filepath"

	"github.com/peterebden/go-cli-init/v5/flags"
	clilogging "github.com/peterebden/go-cli-init/v5/logging"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/generate"
	"github.com/please-build/puku/logging"
	"github.com/please-build/puku/migrate"
	"github.com/please-build/puku/please"
	"github.com/please-build/puku/watch"
	"github.com/please-build/puku/work"
)

var opts = struct {
	Usage     string
	Verbosity clilogging.Verbosity `short:"v" long:"verbosity" description:"Verbosity of output (error, warning, notice, info, debug)" default:"info"`

	Fmt struct {
		Args struct {
			Paths []string `positional-arg-name:"packages" description:"The packages to process"`
		} `positional-args:"true"`
	} `command:"fmt" description:"Format build files in the provided paths"`
	Lint struct {
		Args struct {
			Paths []string `positional-arg-name:"packages" description:"The packages to process"`
		} `positional-args:"true"`
	} `command:"lint" description:"Lint build files in the provided paths"`
	Watch struct {
		Args struct {
			Paths []string `positional-arg-name:"packages" description:"The packages to process"`
		} `positional-args:"true"`
	} `command:"watch" description:"Watch build files in the provided paths and update them when needed"`
	Migrate struct {
		Write bool `short:"w" long:"write" description:"Whether to write the files back or just print them to stdout"`
		Args  struct {
			Paths []string `positional-arg-name:"packages" description:"The packages to process"`
		} `positional-args:"true"`
	} `command:"migrate" description:"Migrates from go_module to go_repo"`
}{
	Usage: `
puku is a tool used to generate and update Go targets in build files
`,
}

var log = logging.GetLogger()

var funcs = map[string]func(conf *config.Config, plzConf *please.Config, orignalWD string) int{
	"fmt": func(conf *config.Config, plzConf *please.Config, orignalWD string) int {
		paths := work.MustExpandPaths(orignalWD, opts.Fmt.Args.Paths)
		if err := generate.NewUpdate(true, plzConf).Update(paths...); err != nil {
			log.Fatalf("%v", err)
		}
		return 0
	},
	"lint": func(conf *config.Config, plzConf *please.Config, orignalWD string) int {
		paths := work.MustExpandPaths(orignalWD, opts.Lint.Args.Paths)
		if err := generate.NewUpdate(false, plzConf).Update(paths...); err != nil {
			log.Fatalf("%v", err)
		}
		return 0
	},
	"watch": func(conf *config.Config, plzConf *please.Config, orignalWD string) int {
		paths := work.MustExpandPaths(orignalWD, opts.Watch.Args.Paths)
		if err := generate.NewUpdate(true, plzConf).Update(paths...); err != nil {
			log.Fatalf("%v", err)
		}

		if err := watch.Watch(plzConf, paths...); err != nil {
			log.Fatalf("%v", err)
		}
		return 0
	},
	"migrate": func(conf *config.Config, plzConf *please.Config, orignalWD string) int {
		paths := work.MustExpandPaths(orignalWD, opts.Migrate.Args.Paths)
		if err := migrate.New(conf, plzConf).Migrate(opts.Migrate.Write, paths...); err != nil {
			log.Fatalf("%v", err)
		}
		return 0
	},
}

func main() {
	flags.ParseFlagsOrDie("puku", &opts, nil)
	logging.InitLogging(opts.Verbosity)

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get wd: %v", err)
	}

	root, err := work.FindRoot()
	if err != nil {
		log.Fatalf("%v", err)
	}

	wd, err = filepath.Rel(root, wd)
	if err != nil {
		log.Fatalf("failed to get wd: %v", err)
	}

	if err := os.Chdir(root); err != nil {
		log.Fatalf("failed to set working dir to repo root: %v", err)
	}

	conf, err := config.ReadConfig(".")
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	plzConf, err := please.QueryConfig(conf.GetPlzPath())
	if err != nil {
		log.Fatalf("failed to query config: %w", err)
	}

	cmd := flags.ParseFlagsOrDie("puku", &opts, nil)
	os.Exit(funcs[cmd](conf, plzConf, wd))
}
