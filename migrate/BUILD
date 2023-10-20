go_library(
    name = "migrate",
    srcs = ["migrate.go"],
    visibility = ["//:all"],
    deps = [
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "///third_party/go/github.com_bazelbuild_buildtools//labels",
        "///third_party/go/golang.org_x_mod//semver",
        "//config",
        "//edit",
        "//generate",
        "//graph",
        "//please",
    ],
)

go_test(
    name = "migrate_test",
    srcs = ["migrate_test.go"],
    deps = [
        ":migrate",
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "//graph",
        "//third_party/go:testify",
    ],
)
