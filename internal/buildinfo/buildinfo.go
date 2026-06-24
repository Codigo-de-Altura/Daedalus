// Package buildinfo exposes static identification for the Daedalus binary.
//
// It is intentionally tiny and free of product logic so every layer of the
// core can depend on it without coupling. Version can be overridden at build
// time via -ldflags "-X github.com/Codigo-de-Altura/Daedalus/internal/buildinfo.Version=<v>".
package buildinfo

// Name is the canonical binary/product name.
const Name = "daedalus"

// Version is the build version of Daedalus. It defaults to a development
// marker and is meant to be injected at release build time.
var Version = "0.1.0-dev"
