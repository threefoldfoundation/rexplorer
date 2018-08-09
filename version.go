package main

import "github.com/rivine/rivine/build"

const (
	rawVersion = "v0.1.1"
)

var (
	version build.ProtocolVersion
)

func init() {
	version = build.MustParse(rawVersion)
}
