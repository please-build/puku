# Puku

Puku is a tool for maintaining your go rules, similar to gazelle. It's named after puku, which is an antelope native to
Southern Africa. This tool is still under development but is rapidly approaching a generally available release!

## Usage

Running `puku fmt` with no args will format all your source files. It has sensible defaults, reading your 
`.plzconfig` file to avoid unnecessary configuration, however it can be configured via `puku.json` files throughout the 
repository. It supports both `go_module()` and `go_repo()` options for third party code, however it can also generate
new `go_repo()` targets to satisfy new dependencies.

Puku can also format files under specific paths using a similar wildcard syntax to Please:

```
$ puku fmt //src/...
```

Puku supports `go_library`, `go_test`, `go_binary`, `go_benchmark`, `proto_library`, and `grpc_library` out of the box, but can be
configured to support other rules. See the configuration section below for more information.

### Migrate

NB: This feature is experimental, and may not always get it right. It should help get started in the migration, but 
there might be a few issues that have to be fixed by hand. You might also be better off deleting your third party 
folder, and regenerating your dependencies with `puku fmt`. 

Use `puku migrate` to migrate your third party rules from `go_module()` to `go_repo`. This subcommand will create
build rules that mimic the behaviour of `go_module()` so this should be a drop in replacement.


### Watch mode

To run puku in watch mode, use `puku watch`. Puku will then watch all directories matched by the wildcards passed, 
and automatically update rules as `.go` sources change.

### No-write mode

By running `puku lint`, puku will run in a lint-only mode. It will exit without output if everything linted fine,
otherwise, it will print the desired state to stdout. This can be useful to integrate with tools like arcanist that can
prompt users with a preview before applying auto-fixes.

## Configuration

Puku can be configured via `puku.json` files that are loaded as puku walks the directory structure. Configuration values
are overriden as new files are discovered at a deeper level in the source tree.

```yaml
{
  // The directory to load and write third party rules to. If using `go_repo`, puku will update this package to satisfy
  // new imports
  "thirdPartyDir": "third_party/go",
  // The path to the please binary
  "pleasePath": "plz",
  // A mapping between import paths and targets for any special cases that puku doesn't currently support
  "knownTargets": {
    "github.com/some/module": "//third_party/go:module"
  },
  // Any kinds that can satisfy an import. These will be treated much the same way puku treats go_library. Right now
  // puku assumes that the rules will use name, srcs, deps, and visibility arguments in the same way go_library does.
  // Puku will attempt to allocate production (i.e. non-test) sources to these targets.
  "libKinds": {
    "my_go_library": {
        // Any deps that the build definition will add to the target. Puku will avoid adding these dependencies via
        // deps.
        "providedDeps": ["//common/go:some_common_lib"],
        // The visibility of the target if no visibility arg is passed
        "defaultVisibility": ["PUBLIC"]
    },
    "my_proto_library": {
        // Setting this to true indicates to puku that these targets don't operate on Go sources, so it shouldn't try
        // to parse the sources to figure out the dependencies. These targets must still satisfy an import for the
        // package directory they're in.
        "nonGoSources": true
    }
  },
  // These are any kinds that behave like tests. Similar to lib kinds, puku assumes they have arguments that are similar
  // to go_test. Puku will try and assign test sources to targets of this kind.
  "testKinds": {
    "testify_test": {
        // Any deps that the build definition will add to the target. Puku will avoid adding these dependencies via
        // deps.
        "providedDeps": ["//third_party/go:testify"],
    },
  }
  // Again, these are similar to lib and test kinds except they are treated as binary targets. Puku assumes a similar
  // set of arguments to go_binary and will allocate binary sources to these targets.
  "binKinds": {
    "testify_test": {
        // Any deps that the build definition will add to the target. Puku will avoid adding these dependencies via
        // deps.
        "providedDeps": ["//third_party/go:testify"],
    },
  }

  // Setting this to true will stop puku from touching this directory and all directories under it. By default, puku
  // will skip over plz-out and .git, however this can be useful to extend that to other directories.
  "stop": false,
}
```

## Overview of the algorithm

Puku will attempt to allocate any .go files in a directory to rules based on their type. There are 3 types:
library, test, and binary. These sources are allocated to rules based on the rules kind type, as configured in the
`puku.json` files.

Once puku has determined the kind type for each source, it will parse the BUILD file to discover the existing build
rules. It will parse the `srcs` arguments of each rule, evaluating `glob()`s as necessary in order to determine any
unallocated sources.

Sources are then allocated to existing rules where possible based on their kind type. If no rule can be found, then a
new rule will be created. The kind type that puku chooses for new rules are the built in base types i.e. `go_library`,
`go_test`, and `go_binary`.

Once all sources have been allocated, the imports for each source file are collected and resolved. Puku will resolve
imports in the following order:

1) Known imports as defined in the configuration file
2) Installed packages from `go_module` or `go_repo`
3) If using `go_repo`, by the module package naming convention (run `plz help go_repo` for more information)
4) If using `go_repo`, by checking the go module proxy, or by reading the `go.mod` file

When using `go_repo`, puku will attempt to automatically add new modules to the build graph, updating the existing
modules as necessary.