package main

import "github.com/rivine/rivine/build"

var (
	rawVersion = "v0.2.1"
	version    build.ProtocolVersion
)

func init() {
	version = build.MustParse(rawVersion)
}
