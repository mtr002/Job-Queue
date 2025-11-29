load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "db",
    srcs = [
        "connection.go",
        "store.go",
    ],
    importpath = "github.com/mtr002/Job-Queue/internal/db",
    visibility = ["//visibility:public"],
    deps = [
        "//internal/interfaces",
        "@com_github_lib_pq//:pq",
        "@com_github_pressly_goose_v3//:goose",
    ],
)
