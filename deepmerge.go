// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Deep merge via reflection

// Package merge defines useful functions to deeply merge arbitrary values.
package merge

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

// During deepValueMerge, must keep track of checks that are
// in progress. The comparison algorithm assumes that all
// checks in progress are true when it reencounters them.
// Visited comparisons are stored in a map indexed by visit.
type visit struct {
	a   unsafe.Pointer
	typ reflect.Type
}

// Merges for deep merge using reflected types. The map argument tracks
// comparisons that have already been seen, which allows short circuiting on
// recursive types.
func deepValueMerge(path string, dst, src reflect.Value, visited map[visit]bool, c *Config) error {
	debugf("deepValueMerge %q\n", path)

	if !dst.IsValid() || !src.IsValid() {
		if dst.IsValid() == src.IsValid() {
			return nil
		}
		return errors.New("dst.IsValid() != src.IsValid()")
	}
	if dst.Type() != src.Type() {
		return errors.New(dst.Type().String() + " != " + src.Type().String())
	}

	// We want to avoid putting more in the visited map than we need to.
	// For any possible reference cycle that might be encountered,
	// hard(src) needs to return true for the src type in the cycle,
	// and it's safe and valid to get Value's internal pointer.
	hard := func(src reflect.Value) bool {
		switch src.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice:
			// Nil pointers cannot be cyclic. Avoid putting them in the visited map.
			return !src.IsNil()
		case reflect.Interface:
			return !src.IsNil() && src.CanAddr()
		}
		return false
	}

	if hard(src) {
		// For a Pointer or Map value, we need to check flagIndir,
		// which we do by calling the pointer method.
		// For Slice or Interface, flagIndir is always set,
		// and using v.ptr suffices.
		ptrval := func(v reflect.Value) unsafe.Pointer {
			switch v.Kind() {
			case reflect.Pointer, reflect.Map, reflect.Slice:
				return v.UnsafePointer()
			default:
				return v.Addr().UnsafePointer()
			}
		}

		debugln(src.Type(), src.CanAddr())

		addr := ptrval(src)

		// Short circuit if references are already seen.
		typ := src.Type()
		v := visit{addr, typ}
		if visited[v] {
			debugln("cycle struct")
			// shallow merge
			dst.Set(src)
			return nil
		}

		// Remember for later.
		visited[v] = true
	}

	if fn := c.transformers[dst.Type()]; fn.IsValid() {
		if err, _ := fn.Call([]reflect.Value{dst.Addr(), src})[0].Interface().(error); err != nil {
			return err
		}
		return nil
	}

	switch dst.Kind() {
	case reflect.Array:
		for i := 0; i < dst.Len(); i++ {
			if err := deepValueMerge(fmt.Sprintf("%s[%d]", path, i),
				dst.Index(i), src.Index(i), visited, c); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		if dst.Len() == 0 && (src.Len() == 0 && c.overwriteEmptySlice) {
			dst.Set(src)
			return nil
		}
		if c.appendSlice {
			dst.Set(reflect.AppendSlice(dst, src))
			return nil
		}

		if dst.Len() < src.Len() {
			if src.Len() <= dst.Cap() {
				dst.Set(dst.Slice(0, src.Len()))
			} else {
				s := reflect.MakeSlice(dst.Type(), src.Len(), src.Len())
				reflect.Copy(s, dst)
				dst.Set(s)
			}
		}
		if dst.UnsafePointer() == src.UnsafePointer() {
			return nil
		}

		for i := 0; i < dst.Len() && i < src.Len(); i++ {
			if err := deepValueMerge(fmt.Sprintf("%s[%d]", path, i),
				dst.Index(i), src.Index(i), visited, c); err != nil {
				return err
			}
		}

		// Ensure that all elements in dst are zeroed if src's len shorter than dst.
		if c.overwriteWithEmptyValue {
			for i := src.Len(); i < dst.Len(); i++ {
				dst.Index(i).SetZero()
			}
		}

		return nil
	case reflect.Interface:
		if c.shouldNotDereference {
			if (dst.IsNil() || c.overwrite) && (!src.IsNil() || c.overwriteWithEmptyValue) {
				dst.Set(src)
			}
			return nil
		}
		if dst.IsNil() || src.IsNil() {
			if dst.IsNil() == src.IsNil() {
				return nil
			}

			if src.IsNil() {
				// Ensure the value that dst contains is zeroed.
				if !dst.IsNil() && !dst.Elem().IsZero() && c.overwriteWithEmptyValue {
					dst.Set(reflect.Zero(dst.Elem().Type()))
				}
				return nil
			}

			if dst.IsNil() {
				dst.Set(reflect.Zero(src.Elem().Type()))
			}
		}

		debugln("path:", path)

		if dst.Elem().Type() != src.Elem().Type() && c.overwrite && !c.appendSlice {
			if c.typeCheck {
				return errors.New("overwrite interface value with difference concrete type")
			}
			dst.Set(src.Elem())
			return nil
		}

		de := reflect.New(dst.Elem().Type()).Elem()
		de.Set(dst.Elem())
		if err := deepValueMerge(fmt.Sprintf("%s(%s)", path, dst.Type()), de, src.Elem(), visited, c); err != nil {
			return err
		}
		dst.Set(de)
		return nil
	case reflect.Pointer:
		if c.shouldNotDereference {
			if (dst.IsNil() || c.overwrite) && (!src.IsNil() || c.overwriteWithEmptyValue) {
				dst.Set(src)
			}
			return nil
		}

		if dst.UnsafePointer() == src.UnsafePointer() {
			return nil
		}

		if dst.IsNil() != src.IsNil() {
			if src.IsNil() {
				if !dst.IsNil() && !dst.Elem().IsZero() && c.overwriteWithEmptyValue {
					// Ensure the value that dst points to is zeroed.
					dst.Elem().SetZero()
				}
				return nil
			}
			if dst.IsNil() {
				dst.Set(reflect.New(dst.Type().Elem()))
			}
		}

		return deepValueMerge(fmt.Sprintf("(*%s)", path), dst.Elem(), src.Elem(), visited, c)
	case reflect.Struct:
		var hasExportedField bool
		for i, n := 0, dst.NumField(); i < n; i++ {
			typeOfF := dst.Type().Field(i)
			if !typeOfF.IsExported() && reflect.Struct != typeOfF.Type.Kind() && !typeOfF.Anonymous {
				continue
			}

			hasExportedField = true
			filedPath := fmt.Sprintf("%s.%s", path, typeOfF.Name)
			if err := deepValueMerge(filedPath, dst.Field(i), src.Field(i), visited, c); err != nil {
				return err
			}
		}

		if hasExportedField {
			return nil
		}
	case reflect.Map:
		if dst.IsNil() != src.IsNil() {
			if dst.IsNil() && src.Len() > 0 {
				dst.Set(reflect.MakeMapWithSize(dst.Type(), src.Len()))
			}
		}
		if dst.UnsafePointer() == src.UnsafePointer() {
			return nil
		}
		for it := src.MapRange(); it.Next(); {
			k := it.Key()
			val1 := it.Value()
			val2 := dst.MapIndex(k)

			if !val1.IsValid() {
				continue
			}

			if !val2.IsValid() {
				v := reflect.New(val1.Type()).Elem()
				v.SetZero()
				val2 = v
				debugf("add map key %#v -> %#v\n", k, val1)

			}

			{
				val := reflect.New(val2.Type()).Elem()
				val.Set(val2)
				val2 = val
			}

			if err := deepValueMerge(fmt.Sprintf("%s[%s]", path,
				k.String()), val2, val1, visited, c); err != nil {
				return err
			}
			dst.SetMapIndex(k, val2)
		}

		// Ensure that all keys in dst are deleted if they are not present in src.
		if c.overwriteWithEmptyValue {
			for it := dst.MapRange(); it.Next(); {
				k := it.Key()
				if !src.MapIndex(k).IsValid() {
					dst.SetMapIndex(k, reflect.Value{})
				}
			}
		}
		return nil
	default:
	}

	// Normal merge suffices
	if (dst.IsZero() || c.overwrite) && (!src.IsZero() || c.overwriteWithEmptyValue) {
		debugf("%q %#v -> %#v\n", path, dst, src)
		dst.Set(src)
	}
	return nil
}

// DeepMerge "deeply merge," the contents of src into dst defined as follows.
// Two values of identical type can deeply merge it following cases applies.
// Values of distinct types can not deeply merge.
//
// Array values deeply merge their corresponding elements.
//
// Struct values deeply merge if their corresponding exported fields.
//
// Func values deeply merge if dst is nil and src is not; otherwise they not deeply merge.
//
// Interface values deeply merge they hold concrete values.
//
// Map values deeply merge when all of the following are true:
// either they are the same map object or their corresponding keys
// (matched using Go equality) map to deeply merged values.
//
// Pointer values deeply merge if they are equal using Go's == operator
// or if they point to deeply merged values.
//
// Slice values deeply merge when all of the following are true:
// either they point to the same initial entry of the same underlying array
// (that is, &x[0] == &y[0]) or their corresponding elements (up to length) deeply merged.
//
// Other values - numbers, bools, strings, and channels - deeply merge
// if dst is zero value and src is not, they deeply merged dst = src using Go's = operator.
//
// On the other hand, pointer values are always equal to themselves.
// because they compare equal using Go's == operator, and that
// is a sufficient condition to be deeply merged, regardless of content.
// DeepMerge has been defined so that the same short-cut applies
// to slices and maps: if x and y are the same slice or the same map,
// they are deeply merged regardless of content.
//
// As DeepMerge traverses the data values it may find a cycle. The
// second and subsequent times that DeepMerge compares two pointer
// values that have been merged before, it treats the values as
// merged rather than examining the values to which they point.
// This ensures that DeepMerge terminates.
func DeepMerge(dst, src any, opts ...Option) error {
	debugf("Merge %#v %[1]T\n", dst)

	if dst == nil || src == nil {
		return errors.New("dst or src is nil")
	}

	vdst := reflect.ValueOf(dst)
	vsrc := reflect.ValueOf(src)
	if reflect.Pointer != vdst.Kind() {
		var sliceMerge, mapMerge bool
		switch vdst.Kind() {
		case reflect.Slice:
			sliceMerge = reflect.Slice == vsrc.Kind() && vdst.Len() >= vsrc.Len()
		case reflect.Map:
			mapMerge = !vdst.IsNil() || (reflect.Map == vsrc.Kind() && vdst.Len() == vsrc.Len())
		}
		if !sliceMerge && !mapMerge {
			return errors.New("dst must have kind Pointer")
		}
	}

	if reflect.Pointer == vdst.Kind() {
		vdst = vdst.Elem()
		if reflect.Pointer == vdst.Kind() {
			if vdst.IsNil() {
				p := reflect.New(vdst.Type().Elem())
				debugf("SetPointer %s %p", p.Elem().Type(), p.UnsafePointer())
				vdst.Set(p)
			}
			vdst = vdst.Elem()
		}
	}

	if reflect.Pointer == vsrc.Kind() {
		vsrc = vsrc.Elem()
	}

	if vdst.Type() != vsrc.Type() {
		return errors.New(vdst.Type().String() + " != " + vsrc.Type().String())
	}

	var c Config
	for _, opt := range opts {
		opt.apply(&c)
	}
	return deepValueMerge("", vdst, vsrc, make(map[visit]bool), &c)
}
