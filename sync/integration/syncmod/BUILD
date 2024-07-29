go_test(
    name = "syncmod_test",
    srcs = ["sync_mod_test.go"],
    data = {
        "repo": ["test_repo"],
        "plz": ["//third_party/binary:plz"],
    },
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "///third_party/go/golang.org_x_mod//modfile",
        "//config",
        "//graph",
        "//please",
        "//sync",
        "//options",
    ],
)
