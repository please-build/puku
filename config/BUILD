go_library(
    name = "config",
    srcs = ["config.go"],
    visibility = [
        "//:all",
        "//cmd/puku:all",
        "//e2e/harness:all",
        "//generate:all",
        "//generate/integration/syncmod:all",
        "//graph:all",
        "//migrate:all",
        "//sync:all",
        "//sync/integration/syncmod:all",
        "//work:all",
    ],
    deps = [
        "///third_party/go/github.com_muhammadmuzzammil1998_jsonc//:jsonc",
        "//kinds",
        "//logging",
    ],
)

go_test(
    name = "config_test",
    srcs = ["config_test.go"],
    deps = [
        ":config",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//kinds",
    ],
)
