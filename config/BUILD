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
        "//work:all",
    ],
    deps = ["//kinds"],
)

go_test(
    name = "config_test",
    srcs = ["config_test.go"],
    deps = [
        ":config",
        "//kinds",
        "//third_party/go:testify",
    ],
)
