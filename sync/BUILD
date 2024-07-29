go_library(
    name = "sync",
    srcs = ["sync.go"],
    visibility = [
        "//cmd/puku:all",
        "//generate:all",
        "//sync/integration/syncmod:all",
    ],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_please-build_buildtools//labels",
        "///third_party/go/golang.org_x_mod//modfile",
        "//config",
        "//edit",
        "//graph",
        "//licences",
        "//logging",
        "//please",
        "//proxy",
    ],
)
