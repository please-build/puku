go_library(
    name = "migrate",
    srcs = ["migrate.go"],
    visibility = [
        "//:all",
        "//cmd/puku:all",
    ],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_please-build_buildtools//labels",
        "///third_party/go/golang.org_x_mod//semver",
        "//config",
        "//edit",
        "//generate",
        "//graph",
        "//licences",
        "//please",
        "//proxy",
        "//options",
    ],
)

go_test(
    name = "migrate_test",
    srcs = ["migrate_test.go"],
    deps = [
        ":migrate",
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//edit",
        "//graph",
        "//options",
    ],
)
