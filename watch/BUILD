go_library(
    name = "watch",
    srcs = ["watch.go"],
    visibility = [
        "//:all",
        "//cmd/puku:all",
    ],
    deps = [
        "///third_party/go/github.com_fsnotify_fsnotify//:fsnotify",
        "//generate",
        "//logging",
        "//please",
        "//options",
    ],
)
