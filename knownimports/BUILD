root = f"third_party/go/toolchain/pkg/{CONFIG.OS}_{CONFIG.ARCH}"

genrule(
    name = "go_root_packages",
    srcs = ["//third_party/go:toolchain"],
    outs = ["go_root_packages"],
    cmd = f'find {root} -name "*.a" | sed -e s=^{root}/== | sed -e s="\.a\$"== > $OUT',
)

go_library(
    name = "knownimports",
    srcs = ["known_imports.go"],
    resources = [":go_root_packages"],
    visibility = [
        "//generate:all",
    ],
)
