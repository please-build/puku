go_test(
    name = "generate_test",
    srcs = [
        "build_file_test.go",
        "deps_test.go",
        "generate_test.go",
    ],
    data = ["test_data"],
    deps = [
        ":generate",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)

go_library(
    name = "generate",
    srcs = [
        "build_file.go",
        "deps.go",
        "generate.go",
        "import.go",
        "rule.go",
    ],
    deps = [
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "///third_party/go/github.com_bazelbuild_buildtools//edit",
        "///third_party/go/github.com_peterebden_go-cli-init_v5//logging",
        "//knownimports",
        "//proxy",
    ],
)
