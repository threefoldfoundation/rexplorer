package main

import "github.com/rivine/rivine/build"

var (
	rawVersion = "v0.2.0"
	version    build.ProtocolVersion
)

func init() {
	version = build.MustParse(rawVersion)
}
