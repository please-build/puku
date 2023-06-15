subinclude("///go//build_defs:go")

go_binary(
    name = "puku",
    srcs = ["puku.go"],
    visibility = ["//package:all"],
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

genrule(
    name = "version",
    srcs = ["VERSION"],
    outs = ["version.build_defs"],
    cmd = "echo VERSION = \\\"$(cat $SRCS)\\\" > $OUT",
    visibility = [
        "//package:all",
    ],
)
