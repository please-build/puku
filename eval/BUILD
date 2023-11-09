go_library(
    name = "eval",
    srcs = ["eval.go"],
    visibility = ["//generate:all"],
    deps = [
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "///third_party/go/github.com_bazelbuild_buildtools//labels",
        "//glob",
        "//please",
    ],
)

go_test(
    name = "eval_test",
    srcs = ["eval_test.go"],
    deps = [
        ":eval",
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "//third_party/go:testify",
    ],
)
