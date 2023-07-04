package merge

import "reflect"

// Exported for testing only.

var DeepValueMerge = func(dst, src reflect.Value, opts ...Option) error {
	var c Config
	Options(opts).apply(&c)

	return deepValueMerge("", dst, src, make(map[visit]string), &c)
}

var DeepValueMap = func(dst, src reflect.Value, opts ...Option) error {
	var c Config
	Options(opts).apply(&c)

	return deepValueMap("", dst, src, make(map[visit]string), &c)
}
