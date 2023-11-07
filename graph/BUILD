go_library(
    name = "graph",
    srcs = ["graph.go"],
    visibility = [
        "//generate:all",
        "//generate/integration/syncmod:all",
        "//licences:all",
        "//migrate:all",
        "//modfile:all",
        "//plz-out/tmp/generate/_generate#pkg_info._build/generate:all",
    ],
    deps = [
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "///third_party/go/github.com_bazelbuild_buildtools//labels",
        "//config",
        "//edit",
        "//fs",
    ],
)

go_test(
    name = "graph_test",
    srcs = ["graph_test.go"],
    data = ["//:test_project"],
    deps = [
        ":graph",
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "///third_party/go/github.com_bazelbuild_buildtools//labels",
        "//config",
        "//edit",
        "//third_party/go:testify",
    ],
)
