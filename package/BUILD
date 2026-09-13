subinclude("//:puku_version")

architectures = [
    "darwin_amd64",
    "darwin_arm64",
    "freebsd_amd64",
    "linux_amd64",
    "linux_arm64",
    # Published only from the PeterNeiss fork, for Please's Windows port: the Puku codelab
    # downloads puku-<version>-<os>_<arch>, and upstream publishes no Windows build.
    "windows_amd64",
]

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
