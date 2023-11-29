go_library(
    name = "work",
    srcs = ["work.go"],
    visibility = [
        "//:all",
        "//cmd/puku:all",
        "//generate",
        "//watch",
    ],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//labels",
        "//config",
    ],
)

go_test(
    name = "work_test",
    srcs = ["work_test.go"],
    deps = [
        ":work",
        "//third_party/go:testify",
    ],
)
