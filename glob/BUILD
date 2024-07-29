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
    deps = [
        ":glob",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)
