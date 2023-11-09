go_library(
    name = "edit",
    srcs = ["edit.go"],
    visibility = [
        "//generate:all",
        "//generate/integration/syncmod:all",
        "//graph:all",
        "//licences:all",
        "//migrate:all",
    ],
    deps = [
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "///third_party/go/github.com_bazelbuild_buildtools//edit",
    ],
)

go_test(
    name = "edit_test",
    srcs = ["edit_test.go"],
    deps = [
        ":edit",
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "//third_party/go:testify",
    ],
)
