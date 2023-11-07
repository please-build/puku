go_library(
    name = "logging",
    srcs = ["logging.go"],
    visibility = [
        "//:all",
        "//cmd/puku:all",
        "//generate:all",
        "//plz-out/tmp/generate/_generate#pkg_info._build/generate:all",
        "//watch:all",
    ],
    deps = [
        "///third_party/go/github.com_peterebden_go-cli-init_v5//logging",
        "///third_party/go/gopkg.in_op_go-logging.v1//:go-logging.v1",
    ],
)
