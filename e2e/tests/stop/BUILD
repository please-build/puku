subinclude("//e2e/build_defs")

# Tests that a package with `stop` set doesn't prevent later paths being processed
puku_e2e_test(
    name = "stop_test",
    srcs = ["stop_test.go"],
    repo = "test_repo",
    deps = [
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//e2e/harness",
        "//edit",
    ],
)
