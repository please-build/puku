go_library(
    name = "test_data",
    srcs = [
        "bar.go",
        "symlink.go",
    ],
)

go_test(
    name = "test_data_test",
    srcs = ["bar_test.go"],
    deps = [":test_data"],
)
