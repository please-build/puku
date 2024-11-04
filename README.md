# Puku

Puku is a tool for maintaining your go rules, similar to gazelle. It's named after puku, which is an antelope native to
Southern Africa. This tool is still under development but is rapidly approaching a generally available release!

## Quick start

The easiest way to get started is to install puku with go:
```bash
$ go install github.com/please-build/puku/cmd/puku
```

Then make sure `$GOPATH/bin` is in `$PATH`. This can be done by adding `export PATH=$PATH:$GOROOT/bin:$GOPATH/bin` to 
your `~/.bashrc`, or similar. 

### Running puku with Please

Add a `BuildConfig` and `Alias` to your `.plzconfig`, or for personal usage, your `.plzconfig.local`:
```
[BuildConfig]
Puku-Version = "9.9.9"

[Alias "puku"]
Cmd = run //third_party/binary:puku --
PositionalLabels = true
Desc = A tool to update BUILD files in Go packages 
```

Then add a remote file to your repo under `third_party/binary/BUILD`
```python
remote_file(
    name = "puku",
    url = f"https://github.com/please-build/puku/releases/download/v{CONFIG.PUKU_VERSION}/puku-{CONFIG.PUKU_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    binary = True,
)
```

This enables you to use `plz puku` in place of `puku`. 

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

### Updating and adding third party dependencies with go.mod (optional) 

Puku will attempt to resolve new imports and add `go_repo` rules to satisfy them. This works most of the time, however 
setting `ModFile = //:gomod` in your Go plugin, is far more robust and highly recommended. Without this, you may just 
have to pass in modules via `requirements = ["github.com/example/module"]`, to help resolve imports to the correct module path.

This approach facilitates using standard go tooling i.e. `go get` to resolve dependencies. Puku will then sync new 
dependencies from your `go.mod` to `third_party/go/BUILD` automatically as necessary, or on demand via `puku sync` 

To do this, add a filegroup in your repo root: 

```python
filegroup(
    name = "gomod",
    srcs = ["go.mod"],
    visibility = ["PUBLIC"],
)
```

And configure the Go plugin like so:
```
[Plugin "go"]
Target = //plugins:go
ModFile = //:gomod
...
```

Then when adding a new module, run `go get github.com/foo/bar` and puku will sync this across when you next run 
`puku fmt` or `puku sync`. Updating modules can be done similarly via `go get -u`, and `puku sync`. Puku currently 
does **not** clear out old dependencies no longer found in the `go.mod`. 

### Migration

Use `puku migrate` to migrate your third party rules from `go_module()` to `go_repo`. This subcommand will create
build rules that mimic the behaviour of `go_module()` so this should be a drop in replacement. This command optionally 
takes modules as positional arguments, allowing a piecemeal migration e.g. `puku migrate github.com/example/module`.

### Watch mode

To run puku in watch mode, use `puku watch`. Puku will then watch all directories matched by the wildcards passed, 
and automatically update rules as `.go` sources change.

### Lint mode

By running `puku lint`, puku will run in a lint-only mode. It will exit without output if everything linted fine,
otherwise, it will print the desired state to stdout. This can be useful to integrate with tools like arcanist that can
prompt users with a preview before applying auto-fixes.

## Supporting custom build definitions

Puku treats targets as one of three types: `library`, `binary`, or `test` targets. Sources are allocated to these 
targets based on their type. Targets that are `library` types are additionally used to satisfy imports from other 
targets. 

For example, if you have the following `BUILD` file:

```
my_go_library(
    name = "foo",
    srcs = ["foo.go"],
)

go_test(
    name = "foo_test",
    srcs = ["foo_test.go"],
    deps = [":foo"],
)
```

Puku can be configured to treat custom types as one of these three kinds by adding some configuration to `puku.json` 
files. These can be checked in throughout the repo, and the kinds will apply to all subdirectories. For example, the
following will make puku treat `my_go_library()` the same way it treats `go_library()`.

```
libKinds": {
    "my_go_library": {
        "providedDeps": ["//common/go:some_common_lib"]
    },
}
``` 

Provided deps are assumed to be added to the list of deps provided to the build rule. Puku will avoid adding these when
it sees an import that's satisfied by them. 

### Non-go sources
Puku will try and determine the dependencies of a target by parsing their sources. Sometimes a target produces a go 
package without taking in go sources directly, for example `proto_library()`. If we want to introduce a new protoc 
plugin, say `grcp_gateway()`, we can teach puku about this like so:

```
libKinds": {
    "grcp_gateway": {
        "nonGoSources": true
    },
}
```

When puku sees a target like this:
```
grpc_gateway(
    name = "foo",
    srcs = ["foo.proto"],
)
```

Puku will avoid trying to parse `foo.proto` as a go source, and will not attempt to remove dependencies from the target,
but it will still resolve imports for that path to that target. 

## Configuration

Puku can be configured via `puku.json` files that are loaded as puku walks the directory structure. Configuration values
are overridden as new files are discovered at a deeper level in the source tree.

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

  // Puku will try and add a subinclude for the Go rules if it's not already subincluded. Setting this to false will 
  // disable this behavior. 
  "ensureSubincludes": false,

  // If you have changed the behavior of the default kinds, you may exclude them here so Puku stops treating them as a
  // known kind. This can be useful for cases where you have changed proto_library to output .go files, rather than to 
  // generate the go_library for that package. 
  "excludeBuiltinKinds": ["proto_library"],
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
new rule will be created. The kind type that puku chooses for new rules are the built-in base types i.e. `go_library`,
`go_test`, and `go_binary`.

Once all sources have been allocated, the imports for each source file are collected and resolved. Puku will resolve
imports in the following order:

1) Known imports as defined in the configuration file
2) Installed packages from `go_module` or `go_repo`
3) If using `go_repo`, by the module package naming convention (run `plz help go_repo` for more information)
4) If using `go_repo`, by checking the go module proxy, or by reading the `go.mod` file

When using `go_repo`, puku will attempt to automatically add new modules to the build graph, updating the existing
modules as necessary.

## Contributing

Contributions are more than welcome. Please make sure to raise an issue first, so we can avoid wasted effort. This 
project and it's contributions are licensed under the Apache-2 licence. 
