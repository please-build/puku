go_test(
    name = "trie_test",
    srcs = ["trie_test.go"],
    deps = [
        ":trie",
        "//third_party/go:testify",
    ],
)

go_library(
    name = "trie",
    srcs = ["trie.go"],
    visibility = [
        "//generate:all",
    ],
)
