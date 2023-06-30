package merge

import "reflect"

type Config struct {
	overwrite               bool
	overwriteWithEmptyValue bool
	typeCheck               bool
	shouldNotDereference    bool

	appendSlice         bool
	overwriteEmptySlice bool

	transformers map[reflect.Type]reflect.Value
}

// Option configures for specific behavior of DeepMerge.
type Option interface {
	apply(c *Config)
}

// Options is a list of Option values that also satisfies the Option interface.
type Options []Option

type option func(*Config)

func (opt option) apply(c *Config) { opt(c) }

// WithOverwrite make merge overwrite non-empty dst attributes with non-empty src attributes values.
func WithOverwrite() Option {
	return option(func(c *Config) { c.overwrite = true })
}

// WithOverwriteWithEmptyValue make merge overwrite non-empty dst attributes with empty src attributes values.
func WithOverwriteWithEmptyValue() Option {
	return option(func(c *Config) {
		c.overwrite = true
		c.overwriteWithEmptyValue = true
	})
}

// WithTypeCheck make merge check types while overwriting it (must be used with WithOverwrite).
func WithTypeCheck() Option {
	return option(func(c *Config) { c.typeCheck = true })
}

// WithoutDereference prevents dereferencing pointers when evaluating whether they are empty
// (i.e. a non-nil pointer is never considered empty).
func WithoutDereference() Option {
	return option(func(c *Config) { c.shouldNotDereference = true })
}

// WithAppendSlice make merge append slices instead of overwriting it.
func WithAppendSlice() Option {
	return option(func(c *Config) { c.appendSlice = true })
}

// WithOverwriteEmptySlice will make merge override empty dst slice with empty src slice.
func WithOverwriteEmptySlice() Option {
	return option(func(c *Config) { c.overwriteEmptySlice = true })
}

// WithTransformer adds transformer to merge, allowing to customize the merging of some types.
// The transformer f must be a function "func(dst *T, src T) error"
func WithTransformer(f any) Option {
	return option(func(c *Config) {
		if c.transformers == nil {
			c.transformers = make(map[reflect.Type]reflect.Value)
		}

		vf := reflect.ValueOf(f)
		typeOfF := vf.Type()
		if reflect.Func != typeOfF.Kind() ||
			typeOfF.NumIn() != 2 || reflect.Pointer != typeOfF.In(0).Kind() ||
			typeOfF.In(0).Elem() != typeOfF.In(1) ||
			typeOfF.NumOut() != 1 || reflect.TypeOf(new(error)).Elem() != typeOfF.Out(0) {
			panic(`f must be a function "func(dst *T, src T) error"`)
		}
		typ := vf.Type().In(0).Elem()
		if _, dup := c.transformers[typ]; dup {
			panic("WithTransformer called twice for type " + typ.String())
		}
		c.transformers[typ] = vf
	})
}
