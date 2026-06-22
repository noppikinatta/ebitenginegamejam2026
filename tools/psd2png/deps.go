//go:build tools
// +build tools

// This file exists only to keep github.com/oov/psd in go.mod/go.sum. The actual
// tool (main.go) uses //go:build ignore so `go mod tidy` does not see its
// imports; without this anchor the dependency would be pruned and
// `go run tools/psd2png/main.go` would fail to build.
package main

import _ "github.com/oov/psd"
