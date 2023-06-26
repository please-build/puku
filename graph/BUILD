go_library(
    name = "graph",
    srcs = ["graph.go"],
    visibility = [
        "//generate:all",
        "//migrate:all",
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
        "//third_party/go:testify",
    ],
)
