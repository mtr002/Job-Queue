load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "lib",
    srcs = ["main.go"],
    importpath = "github.com/mtr002/Job-Queue/cmd/server",
    visibility = ["//visibility:private"],
    deps = [
        "//internal/api",
        "//internal/db",
        "//internal/jobs",
        "//internal/worker",
    ],
)

go_binary(
    name = "server",
    embed = [":lib"],
    visibility = ["//visibility:public"],
)
