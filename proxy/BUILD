go_library(
    name = "proxy",
    srcs = ["proxy.go"],
    visibility = [
        "//cmd/puku:all",
        "//generate:all",
        "//licences:all",
        "//plz-out/tmp/generate/_generate#pkg_info._build/generate:all",
    ],
    deps = [
        "///third_party/go/golang.org_x_mod//modfile",
        "///third_party/go/golang.org_x_mod//semver",
    ],
)
