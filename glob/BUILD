go_library(
    name = "glob",
    srcs = ["glob.go"],
    visibility = [
        "//eval:all",
        "//generate",
    ],
)

go_test(
    name = "glob_test",
    srcs = ["glob_test.go"],
    data = ["//:test_project"],
    deps = ["//third_party/go:testify"],
)
