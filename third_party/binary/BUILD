GO_CI_LINT_VERSION = "1.53.2"

remote_file(
    name = "golangci-lint",
    binary = True,
    exported_files = ["golangci-lint-%s-${OS}-${ARCH}/golangci-lint" % GO_CI_LINT_VERSION],
    extract = True,
    hashes = [
        "a4e83f5bfe52f42134c9783aa68ba31104c36e2ad4c221a3c77510dda66ae81c",  # darwin-amd64
        "63f6c3dab479dac20f39d4a91c4a2c019c054968e2d044f03ee47a3e41253623",  # darwin-arm64
        "2298f73b9bc03b88b91fee06c5d519fc7f9d7f328e2c388615bbd7e85a9d6cae",  # linux-amd64
    ],
    url = "https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.tar.gz" % (
        GO_CI_LINT_VERSION,
        GO_CI_LINT_VERSION,
        CONFIG.OS,
        CONFIG.ARCH,
    ),
)

remote_file(
    name = "plz",
    url = f"https://github.com/thought-machine/please/releases/download/v17.1.0/please_17.1.0_{CONFIG.OS}_{CONFIG.ARCH}",
    binary = True,
    visibility = ["PUBLIC"],
    test_only = True,
)