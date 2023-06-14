subinclude("///go//build_defs:go")

go_binary(
    name = "puku",
    srcs = ["puku.go"],
    deps = [
        "///third_party/go/github.com_peterebden_go-cli-init_v5//flags",
        "///third_party/go/github.com_peterebden_go-cli-init_v5//logging",
        "//config",
        "//generate",
        "//please",
        "//watch",
        "//work",
    ],
)

filegroup(
    name = "test_project",
    srcs = ["test_project"],
    test_only = True,
    visibility = ["PUBLIC"],
)
