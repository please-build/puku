subinclude("//:puku_version")

go_binary(
    name = "puku",
    srcs = ["puku.go"],
    definitions = {
        "github.com/please-build/puku/version.PukuVersion": PUKU_VERSION,
    },
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_peterebden_go-cli-init_v5//flags",
        "///third_party/go/github.com_peterebden_go-cli-init_v5//logging",
        "//config",
        "//generate",
        "//graph",
        "//licences",
        "//logging",
        "//migrate",
        "//options",
        "//please",
        "//proxy",
        "//sync",
        "//version",
        "//watch",
        "//work",
    ],
)
