go_library(
    name = "edit",
    srcs = ["edit.go"],
    visibility = [
        "//e2e/codegen:all",
        "//e2e/tests/codegen:all",
        "//eval:all",
        "//generate:all",
        "//generate/integration/syncmod:all",
        "//graph:all",
        "//licences:all",
        "//migrate:all",
        "//sync:all",
    ],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_please-build_buildtools//edit",
    ],
)

go_test(
    name = "edit_test",
    srcs = ["edit_test.go"],
    deps = [
        ":edit",
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)
