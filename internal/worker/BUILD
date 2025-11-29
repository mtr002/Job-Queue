load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "worker",
    srcs = ["pool.go"],
    importpath = "github.com/mtr002/Job-Queue/internal/worker",
    visibility = ["//visibility:public"],
    deps = [
        "//internal/interfaces",
        "//internal/jobs",
    ],
)