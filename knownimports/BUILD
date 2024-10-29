genrule(
    name = "go_root_packages",
    srcs = ["//third_party/go:std"],
    outs = ["go_root_packages"],
    cmd = 'find $SRCS -name "*.a" | sed -re s=^third_party/go/std/pkg/[^/]+_[^/]+/== | sed -e s="\.a\$"== > $OUT',
)

go_library(
    name = "knownimports",
    srcs = ["known_imports.go"],
    resources = [":go_root_packages"],
    visibility = [
        "//generate:all",
    ],
)
