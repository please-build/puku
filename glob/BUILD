go_library(
    name = "glob",
    srcs = ["glob.go"],
    visibility = [
        "//generate",
        "//plz-out/tmp/generate/_generate#pkg_info._build/generate:all",
    ],
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
