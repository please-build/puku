subinclude("///go//build_defs:go")

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
        "//cmd/puku:all",
        "//package:all",
    ],
)

filegroup(
    name = "mod",
    srcs = ["go.mod"],
    visibility = ["PUBLIC"],
)
