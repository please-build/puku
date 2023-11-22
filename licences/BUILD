go_library(
    name = "licences",
    srcs = ["licences.go"],
    visibility = [
        "//cmd/puku:all",
        "//generate:all",
    ],
    deps = [
        "///third_party/go/github.com_google_go-licenses//licenses",
        "///third_party/go/github.com_google_licenseclassifier_v2//assets",
        "///third_party/go/github.com_please-build_buildtools//build",
        "//edit",
        "//graph",
        "//please",
        "//proxy",
    ],
)

go_test(
    name = "licences_test",
    srcs = ["licences_test.go"],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "//edit",
        "//graph",
        "//proxy",
        "//third_party/go:testify",
    ],
)
