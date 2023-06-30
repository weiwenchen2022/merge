//go:build debug
// +build debug

package merge

import "log"

var (
	debug   = log.Print
	debugf  = log.Printf
	debugln = log.Println
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime | log.Lmicroseconds)
}
