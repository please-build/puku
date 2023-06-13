go_library(
    name = "generate",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = [
        "//:all",
        "//watch",
    ],
    deps = [
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "///third_party/go/github.com_bazelbuild_buildtools//edit",
        "///third_party/go/github.com_peterebden_go-cli-init_v5//logging",
        "//config",
        "//glob",
        "//knownimports",
        "//proxy",
        "//trie",
        "//work",
    ],
)

go_test(
    name = "generate_test",
    srcs = glob(include = ["*_test.go"]),
    data = ["test_data"],
    deps = [
        ":generate",
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "//third_party/go:testify",
        "//trie",
    ],
)
