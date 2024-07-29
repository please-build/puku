go_library(
    name = "eval",
    srcs = ["eval.go"],
    visibility = ["//generate:all"],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_please-build_buildtools//labels",
        "//glob",
        "//please",
    ],
)

go_test(
    name = "eval_test",
    srcs = ["eval_test.go"],
    data = ["//:test_project"],
    deps = [
        ":eval",
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//glob",
    ],
)
