subinclude("///go//build_defs:go")

go_toolchain(
    name = "toolchain",
    version = CONFIG.GO_VERSION,
    install_std = False,
)

go_stdlib(
    name = "std",
    tags = [
        "netgo",
        "osusergo",
    ],
)

go_repo(
    licences = ["ISC"],
    module = "github.com/davecgh/go-spew",
    version = "v1.1.1",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/xerrors",
    version = "v0.0.0-20200804184101-5ec99f83aff1",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/prometheus/client_model",
    version = "v0.0.0-20190812154241-14fe0d1b01d4",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/golang/mock",
    version = "v1.1.1",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "github.com/pmezard/go-difflib",
    version = "v1.0.0",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "github.com/golang/protobuf",
    version = "v1.5.3",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/term",
    version = "v0.16.0",
)

go_repo(
    licences = ["BSD-2-Clause"],
    module = "gopkg.in/check.v1",
    version = "v0.0.0-20161208181325-20d25e280405",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "google.golang.org/protobuf",
    version = "v1.32.0",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/chzyer/logex",
    version = "v1.1.10",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/chzyer/test",
    version = "v0.0.0-20180213035817-a1ea475d72b1",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/lint",
    version = "v0.0.0-20190313153728-d0100b6bd8b3",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/mod",
    version = "v0.14.0",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "github.com/thought-machine/go-flags",
    version = "v1.6.3",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/crypto",
    version = "v0.0.0-20190308221718-c2843e01d9a2",
)

go_repo(
    licences = [
        "MIT",
        "BSD-3-Clause",
    ],
    module = "honnef.co/go/tools",
    version = "v0.0.0-20190523083050-ea95bdfd59fc",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "gopkg.in/op/go-logging.v1",
    version = "v1.0.0-20160211212156-b2cb9fa56473",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "go.starlark.net",
    version = "v0.0.0-20210223155950-e043a3d3c984",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "google.golang.org/genproto",
    version = "v0.0.0-20200526211855-cb27e3aa2013",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/envoyproxy/protoc-gen-validate",
    version = "v0.1.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/peterebden/go-cli-init/v5",
    version = "v5.2.1",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/sys",
    version = "v0.16.0",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "github.com/google/go-cmp",
    version = "v0.5.1",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/chzyer/readline",
    version = "v0.0.0-20180603132655-2972be24d48e",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/net",
    version = "v0.20.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/census-instrumentation/opencensus-proto",
    version = "v0.2.1",
)

go_repo(
    name = "testify",
    install = ["..."],
    licences = ["MIT"],
    module = "github.com/stretchr/testify",
    version = "v1.8.4",
    visibility = [
        "//config:all",
        "//e2e/codegen:all",
        "//e2e/syncmod:all",
        "//e2e/tests/codegen:all",
        "//edit:all",
        "//eval:all",
        "//generate:all",
        "//generate/integration/syncmod:all",
        "//glob:all",
        "//graph:all",
        "//licences:all",
        "//migrate:all",
        "//sync/integration/syncmod:all",
        "//trie:all",
        "//work:all",
    ],
)

go_mod_download(
    name = "build_tools_dl",
    licences = ["Apache-2.0"],
    module = "github.com/peterebden/buildtools",
    version = "v0.0.0-20240111140234-77ffe55926d9",
)

go_repo(
    download = ":build_tools_dl",
    module = "github.com/please-build/buildtools",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/golang/glog",
    version = "v0.0.0-20160126235308-23def4e6c14b",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/stretchr/objx",
    version = "v0.5.1",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/tools",
    version = "v0.17.0",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/text",
    version = "v0.3.7",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/sync",
    version = "v0.6.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "google.golang.org/grpc",
    version = "v1.27.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/envoyproxy/go-control-plane",
    version = "v0.9.1-0.20191026205805-5f8ba28d4473",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/oauth2",
    version = "v0.0.0-20180821212333-d2e6202438be",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/yuin/goldmark",
    version = "v1.4.13",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/dustin/go-humanize",
    version = "v1.0.1",
)

go_repo(
    licences = ["MIT"],
    module = "gopkg.in/yaml.v3",
    version = "v3.0.1",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "github.com/fsnotify/fsnotify",
    version = "v1.7.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/google/go-licenses",
    version = "v1.6.1-0.20230903011517-706b9c60edd4",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/go-logr/logr",
    version = "v1.4.1",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/golang/groupcache",
    version = "v0.0.0-20210331224755-41bb18bfe9da",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/google/licenseclassifier/v2",
    version = "v2.0.0",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/sergi/go-diff",
    version = "v1.3.1",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "go.opencensus.io",
    version = "v0.24.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "k8s.io/klog/v2",
    version = "v2.120.1",
)
