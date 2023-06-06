go_library(
    name = "generate",
    srcs = [
        "build_file.go",
        "deps.go",
        "generate.go",
        "import.go",
        "rule.go",
    ],
    visibility = ["//:all"],
    deps = [
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "///third_party/go/github.com_bazelbuild_buildtools//edit",
        "//knownimports",
        "//proxy",
    ],
)

go_test(
    name = "generate_test",
    srcs = ["generate_test.go"],
    deps = [
        ":generate",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
