go_library(
    name = "trie",
    srcs = ["trie.go"],
    visibility = [
        "//generate:all",
    ],
)

go_test(
    name = "trie_test",
    srcs = ["trie_test.go"],
    deps = [
        ":trie",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
