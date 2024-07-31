go_library(
    name = "licences",
    srcs = ["licences.go"],
    visibility = [
        "//cmd/puku:all",
        "//generate:all",
        "//migrate:all",
        "//sync:all",
        "//sync/integration/syncmod:all",
    ],
    deps = [
        "///third_party/go/github.com_google_go-licenses//licenses",
        "///third_party/go/github.com_google_licenseclassifier_v2//assets",
        "///third_party/go/github.com_please-build_buildtools//build",
        "//edit",
        "//graph",
        "//proxy",
    ],
)

go_test(
    name = "licences_test",
    srcs = ["licences_test.go"],
    deps = [
        ":licences",
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//edit",
        "//graph",
        "//proxy",
        "//options",
    ],
)
