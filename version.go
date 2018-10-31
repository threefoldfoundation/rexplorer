package main

import "github.com/rivine/rivine/build"

var (
	rawVersion = "v0.3.0"
	version    build.ProtocolVersion
)

func init() {
	version = build.MustParse(rawVersion)
}
