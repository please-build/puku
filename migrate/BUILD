go_library(
    name = "migrate",
    srcs = ["migrate.go"],
    visibility = [
        "//:all",
        "//cmd/puku:all",
    ],
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
