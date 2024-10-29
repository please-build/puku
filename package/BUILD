subinclude("//:puku_version")

filegroup(
    name = "architectures",
    srcs = ["architectures.build_defs"],
    visibility = ["PUBLIC"],
)

subinclude(":architectures")

def cross_compile(version, arch):
    return build_rule(
        name = f"puku_{arch}",
        srcs = [f"///{arch}//cmd/puku"],
        outs = [f"puku-{version}-{arch}"],
        cmd = "mv $SRC $OUT",
        labels = ["manual"],
    )

filegroup(
    name = f"release_files",
    srcs = [cross_compile(PUKU_VERSION, arch) for arch in architectures],
    labels = [
        f"hlink:plz-out/package",
        "manual",
    ],
)
