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
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_please-build_buildtools//labels",
        "//config",
        "//edit",
        "//eval",
        "//fs",
        "//glob",
        "//graph",
        "//kinds",
        "//knownimports",
        "//licences",
        "//logging",
        "//please",
        "//proxy",
        "//trie",
        "//options",
    ],
)

testify_test(
    name = "generate_test",
    srcs = glob(["*_test.go"]),
    data = ["//:test_project"],
    deps = [
        ":generate",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//config",
        "//edit",
        "//kinds",
        "//please",
        "//proxy",
        "//trie",
        "//options",
    ],
)
