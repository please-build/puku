go_library(
    name = "graph",
    srcs = ["graph.go"],
    visibility = [
        "//generate:all",
        "//generate/integration/syncmod:all",
        "//licences:all",
        "//migrate:all",
        "//modfile:all",
        "//cmd/puku:all",
        "//sync:all",
        "//sync/integration/syncmod:all",
    ],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_please-build_buildtools//labels",
        "//config",
        "//edit",
        "//fs",
        "//logging",
    ],
)

go_test(
    name = "graph_test",
    srcs = ["graph_test.go"],
    data = ["//:test_project"],
    deps = [
        ":graph",
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_please-build_buildtools//labels",
        "//config",
        "//edit",
        "//third_party/go:testify",
    ],
)
