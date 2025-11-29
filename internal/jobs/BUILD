load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "jobs",
    srcs = [
        "job.go",
        "manager.go",
    ],
    importpath = "github.com/mtr002/Job-Queue/internal/jobs",
    visibility = ["//visibility:public"],
    deps = [
        "//internal/interfaces",
        "@com_github_google_uuid//:uuid",
    ],
)