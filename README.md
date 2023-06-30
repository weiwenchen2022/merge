# Merge

package merge defines useful functions to deeply merge arbitrary values in Golang. Useful for configuration default values, avoiding messy if-statements.

Merge merges same-type values by setting default values in zero-value fields. Merge won't merge unexported (private) fields. It will do recursively any exported one.

## Install

`go get github.com/weiwenchen2022/merge`

## Reference

GoDoc: [http://godoc.org/github.com/weiwenchen2022/merge](http://godoc.org/github.com/weiwenchen2022/merge)
