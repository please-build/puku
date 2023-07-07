go_test(
    name = "syncmod_test",
    srcs = ["sync_mod_test.go"],
    data = {
        "repo": ["test_repo"],
    },
    deps = [
        "///third_party/go/golang.org_x_mod//modfile",
        "//config",
        "//generate",
        "//graph",
        "//please",
        "//third_party/go:testify",
    ],
)
