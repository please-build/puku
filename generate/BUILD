subinclude("//build_defs:testify_test")

go_library(
    name = "generate",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = [
        "//:all",
        "//cmd/puku:all",
        "//generate/integration/syncmod:all",
        "//migrate:all",
        "//watch",
    ],
    deps = [
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "///third_party/go/github.com_bazelbuild_buildtools//labels",
        "///third_party/go/golang.org_x_mod//modfile",
        "//config",
        "//edit",
        "//fs",
        "//glob",
        "//graph",
        "//kinds",
        "//knownimports",
        "//logging",
        "//please",
        "//proxy",
        "//trie",
    ],
)

testify_test(
    name = "generate_test",
    srcs = glob(["*_test.go"]),
    data = ["//:test_project"],
    deps = [
        ":generate",
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "//config",
        "//edit",
        "//graph",
        "//kinds",
        "//please",
        "//proxy",
        "//trie",
    ],
)
