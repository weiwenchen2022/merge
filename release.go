//go:build !debug
// +build !debug

package merge

var (
	debug   = func(...any) {}
	debugf  = func(string, ...any) {}
	debugln = func(...any) {}
)
