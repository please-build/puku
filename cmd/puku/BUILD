subinclude("//:version")

go_binary(
    name = "puku",
    srcs = ["puku.go"],
    definitions = {
        "github.com/please-build/puku/version.PukuVersion": VERSION,
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
        "//please",
        "//proxy",
        "//sync",
        "//version",
        "//watch",
        "//work",
        "//options",
    ],
)
