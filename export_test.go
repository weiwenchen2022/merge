package merge

import "reflect"

// Exported for testing only.

var DeepValueMerge = func(dst, src reflect.Value, opts ...Option) error {
	var c Config
	for _, opt := range opts {
		opt.apply(&c)
	}
	return deepValueMerge("", dst, src, map[visit]bool{}, &c)
}
