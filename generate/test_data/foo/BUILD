go_test(
    name = "foo_test",
    srcs = ["external_test.go"],
    external = True,
    deps = ["//third_party/go:testify"],
)

go_library(
    name = "foo",
    srcs = ["foo.go"],
)

go_test(
    name = "foo_test",
    srcs = ["foo_test.go"],
    deps = [
        ":foo",
        "//third_party/go:testify",
    ],
)
