subinclude("///go//build_defs:go")

filegroup(
    name = "test_project",
    srcs = ["test_project"],
    test_only = True,
    visibility = ["PUBLIC"],
)

genrule(
    name = "puku_version",
    srcs = ["PUKU_VERSION"],
    outs = ["puku_version.build_defs"],
    cmd = "echo PUKU_VERSION = \\\"$(cat $SRCS)\\\" > $OUT",
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
