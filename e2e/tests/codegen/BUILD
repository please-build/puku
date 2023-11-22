subinclude("//e2e/build_defs")

# Tests when the srcs of a target is generated i.e. is a build label rather than a file on disk
puku_e2e_test(
    name = "codegen_test",
    srcs = ["codgen_test.go"],
    repo = "test_repo",
    deps = [
        "//e2e/harness",
        "//edit",
        "//third_party/go:testify",
    ],
)
