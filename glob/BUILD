go_library(
    name = "glob",
    srcs = ["glob.go"],
    visibility = ["//generate"],
)

go_test(
    name = "glob_test",
    srcs = ["glob_test.go"],
    data = ["//:test_project"],
    deps = [
        ":glob",
        "//third_party/go:testify",
    ],
)
