// Package `bcpfs-perms/v` provides the version info that is injected by the
// linker via `-X package/varname=value` flags.
package v

// `Version` and `Build` are injected by the `Makefile`.
var (
	Version string
	Build   string
)
