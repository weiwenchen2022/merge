// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Deep map via reflection

package merge

import (
	"errors"
	"fmt"
	"reflect"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// Maps for deep map using reflected types. The map argument tracks
// comparisons that have already been seen, which allows short circuiting on
// recursive types.
func deepValueMap(path string, dst, src reflect.Value, visited map[visit]string, c *Config) error {
	// debugf("deepValueMap %q\n", path)

	if !dst.IsValid() || !src.IsValid() {
		if dst.IsValid() == src.IsValid() {
			return nil
		}
		return errors.New("v1.IsValid() != v2.IsValid()")
	}

	// if dst.Type() != src.Type() {
	// 	return errors.New(dst.Type().String() + " != " + src.Type().String())
	// }

	// We want to avoid putting more in the visited map than we need to.
	// For any possible reference cycle that might be encountered,
	// hard(v) needs to return true for the src type in the cycle,
	// and it's safe and valid to get Value's internal pointer.
	hard := func(v reflect.Value) bool {
		switch v.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice:
			// Nil pointers cannot be cyclic. Avoid putting them in the visited map.
			return !v.IsNil()
		case reflect.Interface:
			return !v.IsNil() && v.CanAddr()
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

		addr := ptrval(src)

		// Short circuit if references are already seen.
		typ := src.Type()
		v := visit{addr, typ}
		if visited[v] != "" {
			debugln("cycle traverses. conflicts are:\nA) " + visited[v] + "\n\nand\nB) " + stack())
			// shallow map
			dst.Set(src)
			return nil
		}

		// Remember for later.
		visited[v] = stack()
	}

	if fn := c.transformers[dst.Type()]; fn.IsValid() {
		if err, _ := fn.Call([]reflect.Value{dst.Addr(), src})[0].Interface().(error); err != nil {
			return err
		}
		return nil
	}

	switch dst.Kind() {
	case reflect.Array:
		switch src.Kind() {
		case reflect.String:
			if reflect.Uint8 != dst.Type().Elem().Kind() {
				break
			}

			fallthrough
		case reflect.Array, reflect.Slice:
			for i := 0; i < dst.Len() && i < src.Len(); i++ {
				if err := deepValueMap(fmt.Sprintf("%s[%d]", path, i),
					dst.Index(i), src.Index(i), visited, c); err != nil {
					return err
				}
			}
			return nil
		}
	case reflect.Slice:
		de := dst.Type().Elem()
		switch src.Kind() {
		case reflect.Slice, reflect.Array:
		case reflect.String:
			if reflect.Uint8 == de.Kind() {
				break
			}
			fallthrough
		default:
			return errors.New("src must have kind Slice or Array")
		}

		if dst.Len() == 0 && (src.Len() == 0 && c.overwriteEmptySlice) {
			if reflect.Slice == src.Kind() {
				if dst.IsNil() != src.IsNil() {
					if dst.Type() == src.Type() {
						dst.Set(src)
					} else {
						if src.IsNil() {
							dst.Set(reflect.Zero(dst.Type()))
						} else {
							dst.Set(reflect.MakeSlice(dst.Type(), 0, 0))
						}
					}
				}
			}
			return nil
		}

		if c.appendSlice {
			var ss reflect.Value
			sk := src.Kind()
			switch sk {
			case reflect.String:
				ss = reflect.MakeSlice(reflect.SliceOf(de), src.Len(), src.Len())
				for i := 0; i < src.Len(); i++ {
					ss.Index(i).Set(src.Index(i).Convert(de))
				}
			case reflect.Slice, reflect.Array:
				se := src.Type().Elem()
				if de == se {
					if reflect.Array == sk && src.CanAddr() {
						ss = src.Slice(0, src.Len())
					} else {
						ss = src
					}
				}
				if ss.IsValid() {
					break
				}

				if !se.AssignableTo(de) && !se.ConvertibleTo(de) {
					return errors.New("src element type can not convertible to dst element type")
				}

				ss = reflect.MakeSlice(reflect.SliceOf(de), src.Len(), src.Len())
				for i := 0; i < src.Len(); i++ {
					ss.Index(i).Set(src.Index(i).Convert(de))
				}
			}

			dst.Set(reflect.AppendSlice(dst, ss))
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
			if err := deepValueMap(fmt.Sprintf("%s[%d]", path, i),
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
			if (dst.IsNil() || c.overwrite) &&
				(reflect.Interface == src.Kind() && (!src.IsNil() || c.overwriteWithEmptyValue)) {
				dt := dst.Type()
				st := src.Type()
				if dt == st {
					dst.Set(src)
				} else if st.ConvertibleTo(dt) {
					dst.Set(src.Convert(dt))
				}
			}
			return nil
		}

		var se reflect.Value
		switch src.Kind() {
		default:
			se = src
		case reflect.Interface:
			if dst.IsNil() || src.IsNil() {
				if dst.IsNil() == src.IsNil() {
					return nil
				}
			}

			if src.IsNil() {
				// Ensure the value that dst contains is zeroed.
				if !dst.IsNil() && !dst.Elem().IsZero() && c.overwriteWithEmptyValue {
					dst.Set(reflect.Zero(dst.Elem().Type()))
				}
				return nil
			}

			se = src.Elem()
		}

		var de reflect.Value
		if dst.IsNil() {
			de = reflect.New(se.Type()).Elem()
		} else {
			de = reflect.New(dst.Elem().Type()).Elem()
			de.Set(dst.Elem())
		}

		if de.Kind() != se.Kind() {
			if c.overwrite && !c.appendSlice {
				if !se.Type().Implements(dst.Type()) {
					return errors.New("overwrite src type not implements dst interface type")
				}
				if de.Type() != se.Type() && c.typeCheck {
					return errors.New("overwrite interface value with difference concrete type")
				}

				dst.Set(se)
			}
			return nil
		}

		if err := deepValueMap(fmt.Sprintf("%s(%s)", path, dst.Type()), de, se, visited, c); err != nil {
			return err
		}
		dst.Set(de)
		return nil
	case reflect.Pointer:
		if c.shouldNotDereference {
			if (dst.IsNil() || c.overwrite) &&
				(reflect.Pointer == src.Kind() && (!src.IsNil() || c.overwriteWithEmptyValue)) {
				dt := dst.Type()
				st := src.Type()
				if dt == st {
					dst.Set(src)
				} else if st.ConvertibleTo(dt) {
					dst.Set(src.Convert(dt))
				}
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

		var se reflect.Value
		switch src.Kind() {
		default:
			se = src
		case reflect.Pointer:
			se = src.Elem()
		}

		return deepValueMap(fmt.Sprintf("(*%s)", path), dst.Elem(), se, visited, c)
	case reflect.Struct:
		switch src.Kind() {
		case reflect.Pointer:
			src = src.Elem()
		}

		switch src.Kind() {
		default:
			return fmt.Errorf("%s cannot be represents %s", dst.Kind().String(), src.Kind().String())
		case reflect.Map:
			var hasExportedField bool
			for i, n := 0, dst.NumField(); i < n; i++ {
				typeOfF := dst.Type().Field(i)
				if !typeOfF.IsExported() {
					continue
				}

				hasExportedField = true

				fieldName := typeOfF.Name
				k := reflect.ValueOf(fieldName)
				se := src.MapIndex(k)
				if !se.IsValid() {
					r, size := utf8.DecodeRuneInString(fieldName)
					fieldName = string(unicode.ToLower(r)) + fieldName[size:]
					k = reflect.ValueOf(fieldName)
					se = src.MapIndex(k)
				}
				if !se.IsValid() {
					continue
				}

				se = reflect.ValueOf(se.Interface())

				fieldPath := fmt.Sprintf("%s[%s]", path, typeOfF.Name)

				df := dst.Field(i)
				if reflect.Pointer == df.Kind() {
					if df.IsNil() {
						df.Set(reflect.New(df.Type().Elem()))
					}
					df = df.Elem()
				}
				if err := deepValueMap(fieldPath, df, se, visited, c); err != nil {
					return err
				}
			}

			debugln("hasExportedField", hasExportedField)

			if hasExportedField {
				return nil
			}
		case reflect.Struct:
			var hasExportedField bool
			for i := 0; i < dst.NumField() && i < src.NumField(); i++ {
				typeOfF := dst.Type().Field(i)
				if !typeOfF.IsExported() && reflect.Struct != typeOfF.Type.Kind() && !typeOfF.Anonymous {
					continue
				}

				hasExportedField = true
				fieldPath := fmt.Sprintf("%s[%s]", path, typeOfF.Name)
				if err := deepValueMap(fieldPath, dst.Field(i), src.Field(i), visited, c); err != nil {
					return err
				}
			}

			if hasExportedField {
				return nil
			}
		}
	case reflect.Map:
		switch src.Kind() {
		default:
			return fmt.Errorf("%s cannot be represents %s", dst.Kind().String(), src.Kind().String())
		case reflect.Struct:
			for i, n := 0, src.NumField(); i < n; i++ {
				typeOfF := src.Type().Field(i)
				if !typeOfF.IsExported() {
					continue
				}

				fieldName := typeOfF.Name
				k := reflect.ValueOf(fieldName)
				de := dst.MapIndex(k)
				if !de.IsValid() {
					r, size := utf8.DecodeRuneInString(fieldName)
					fieldName = string(unicode.ToLower(r)) + fieldName[size:]
					k = reflect.ValueOf(fieldName)
					de = dst.MapIndex(k)
				}

				if !de.IsValid() {
					de = reflect.New(src.Field(i).Type()).Elem()
				} else {
					de = reflect.ValueOf(de.Interface())
					elm := reflect.New(de.Type()).Elem()
					elm.Set(de)
					de = elm
				}

				if err := deepValueMap(fmt.Sprintf("%s[%s]", path, k),
					de, src.Field(i), visited, c); err != nil {
					return err
				}
				dst.SetMapIndex(k, de)
			}
			return nil
		case reflect.Map:
		}

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
				debugf("add map key (%#v, %#v)\n", k, val1)
			} else {
				v := reflect.New(val2.Type()).Elem()
				v.Set(val2)
				val2 = v
			}

			if err := deepValueMap(fmt.Sprintf("%s[%s]", path,
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
	case reflect.String:
		switch src.Kind() {
		default:
			return fmt.Errorf("%s can not represents %s", dst.Kind().String(), src.Kind().String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if src.Int() != int64(int32(src.Int())) {
				return fmt.Errorf("%d cannot be represented as an int32", src.Int())
			}

			r := reflect.ValueOf(int32(src.Int()))
			s := r.Convert(reflect.TypeOf(""))
			if (dst.IsZero() || c.overwrite) && (!s.IsZero() || c.overwriteWithEmptyValue) {
				if c.typeCheck && c.overwrite {
					if dst.Type() != src.Type() {
						return fmt.Errorf("overwrite two different types %s <- %s", dst.Type(), src.Type())
					}
				}

				debugf("%q (%s, %#v) <- (%s, %#U)\n", path, dst.Type(), dst, src.Type(), src)
				dst.Set(s.Convert(dst.Type()))
			}
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if src.Uint() != uint64(int32(src.Uint())) {
				return fmt.Errorf("%d cannot be represented as an int32", src.Uint())
			}

			r := reflect.ValueOf(int32(src.Uint()))
			s := r.Convert(reflect.TypeOf(""))
			if (dst.IsZero() || c.overwrite) && (!s.IsZero() || c.overwriteWithEmptyValue) {
				if c.typeCheck && c.overwrite {
					if dst.Type() != src.Type() {
						return fmt.Errorf("overwrite two different types %s <- %s", dst.Type(), src.Type())
					}
				}

				debugf("%q (%s, %#v) <- (%s, %#U)\n", path, dst.Type(), dst, src.Type(), src)
				dst.Set(s.Convert(dst.Type()))
			}
			return nil
		case reflect.Slice:
			switch src.Type().Elem().Kind() {
			case reflect.Uint8:
				bs := reflect.ValueOf(unsafe.Slice((*uint8)(src.UnsafePointer()), src.Len()))
				s := bs.Convert(reflect.TypeOf(""))
				if (dst.IsZero() || c.overwrite) && (!s.IsZero() || c.overwriteWithEmptyValue) {
					if c.typeCheck && c.overwrite {
						if dst.Type() != src.Type() {
							return fmt.Errorf("overwrite two different types %s <- %s", dst.Type(), src.Type())
						}
					}

					debugf("%q (%s, %#v) <- (%s, %q)\n", path, dst.Type(), dst, src.Type(), src)
					dst.Set(s.Convert(dst.Type()))
				}
				return nil
			case reflect.Int32:
				rs := reflect.ValueOf(unsafe.Slice((*int32)(src.UnsafePointer()), src.Len()))
				s := rs.Convert(reflect.TypeOf(""))
				if (dst.IsZero() || c.overwrite) && (!s.IsZero() || c.overwriteWithEmptyValue) {
					if c.typeCheck && c.overwrite {
						if dst.Type() != src.Type() {
							return fmt.Errorf("overwrite two different types %s <- %s", dst.Type(), src.Type())
						}
					}

					debugf("%q (%s, %#v) <- (%s, %q)\n", path, dst.Type(), dst, src.Type(), src)
					dst.Set(s.Convert(dst.Type()))
				}
				return nil
			default:
			}

		case reflect.String:
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var i int64
		switch src.Kind() {
		default:
			return fmt.Errorf("%s can not represents %s", dst.Kind().String(), src.Kind().String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i = src.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if src.Uint() != uint64(int64(src.Uint())) {
				return fmt.Errorf("%d cannot be represented as an %s", src.Uint(), dst.Kind().String())
			}
			i = int64(src.Uint())
		case reflect.Float32, reflect.Float64:
			if src.Float() != float64(int64(src.Float())) {
				return fmt.Errorf("%f cannot be represented as an %s", src.Float(), dst.Kind().String())
			}
			i = int64(src.Float())
		case reflect.Complex64, reflect.Complex128:
			if imag(src.Complex()) != 0 {
				return fmt.Errorf("%f cannot be represented as an %s", src.Complex(), dst.Kind().String())
			}

			f := real(src.Complex())
			if f != float64(int64(f)) {
				return fmt.Errorf("%f cannot be represented as an %s", src.Complex(), dst.Kind().String())
			}
			i = int64(f)
		}

		if dst.OverflowInt(i) {
			return fmt.Errorf("%d overflow %s", i, dst.Kind().String())
		}

		if (dst.IsZero() || c.overwrite) && (i != 0 || c.overwriteWithEmptyValue) {
			if c.typeCheck && c.overwrite {
				if dst.Type() != src.Type() {
					return fmt.Errorf("overwrite two different types %s <- %s", dst.Type(), src.Type())
				}
			}

			debugf("%q (%s, %#v) <- (%s, %#v)\n", path, dst.Type(), dst, src.Type(), src)
			dst.SetInt(i)
		}
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var i uint64
		switch src.Kind() {
		default:
			return fmt.Errorf("%s can not represents %s", dst.Kind().String(), src.Kind().String())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			i = src.Uint()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if src.Int() < 0 {
				return fmt.Errorf("%d cannot be represented as an %s", src.Int(), dst.Kind().String())
			}
			i = uint64(src.Int())
		case reflect.Float32, reflect.Float64:
			if src.Float() != float64(uint64(src.Float())) {
				return fmt.Errorf("%f cannot be represented as an %s", src.Float(), dst.Kind().String())
			}
			i = uint64(src.Float())
		case reflect.Complex64, reflect.Complex128:
			if imag(src.Complex()) != 0 {
				return fmt.Errorf("%f cannot be represented as an %s", src.Complex(), dst.Kind().String())
			}

			f := real(src.Complex())
			if f != float64(uint64(f)) {
				return fmt.Errorf("%f cannot be represented as an %s", src.Complex(), dst.Kind().String())
			}
			i = uint64(f)
		}

		if dst.OverflowUint(i) {
			return fmt.Errorf("%d overflow %s", i, dst.Kind().String())
		}

		if (dst.IsZero() || c.overwrite) && (i != 0 || c.overwriteWithEmptyValue) {
			if c.typeCheck && c.overwrite {
				if dst.Type() != src.Type() {
					return fmt.Errorf("overwrite two different types %s <- %s", dst.Type(), src.Type())
				}
			}

			debugf("%q (%s, %#v) <- (%s, %#v)\n", path, dst.Type(), dst, src.Type(), src)
			dst.SetUint(i)
		}

		return nil
	case reflect.Float32, reflect.Float64:
		var f float64
		switch src.Kind() {
		default:
			return fmt.Errorf("%s can not represents %s", dst.Kind().String(), src.Kind().String())
		case reflect.Float32, reflect.Float64:
			f = src.Float()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if src.Int() != int64(float64(src.Int())) {
				return fmt.Errorf("%d cannot be represented as an %s", src.Int(), dst.Kind().String())
			}
			f = float64(src.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if src.Uint() != uint64(float64(src.Uint())) {
				return fmt.Errorf("%d cannot be represented as an %s", src.Uint(), dst.Kind().String())
			}
			f = float64(src.Uint())
		}

		if dst.OverflowFloat(f) {
			return fmt.Errorf("%f overflow %s", f, dst.Kind().String())
		}

		if (dst.IsZero() || c.overwrite) && (f != 0 || c.overwriteWithEmptyValue) {
			if c.typeCheck && c.overwrite {
				if dst.Type() != src.Type() {
					return fmt.Errorf("overwrite two different types %s <- %s", dst.Type(), src.Type())
				}
			}

			debugf("%q (%s, %#v) <- (%s, %#v)\n", path, dst.Type(), dst, src.Type(), src)
			dst.SetFloat(f)
		}

		return nil
	case reflect.Complex64, reflect.Complex128:
		var c1 complex128
		switch src.Kind() {
		default:
			return fmt.Errorf("%s can not represents %s", dst.Kind().String(), src.Kind().String())
		case reflect.Complex64, reflect.Complex128:
			c1 = src.Complex()
		case reflect.Float32, reflect.Float64:
			c1 = complex(src.Float(), 0)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			c1 = complex(float64(src.Int()), 0)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			c1 = complex(float64(src.Uint()), 0)
		}

		if dst.OverflowComplex(c1) {
			return errors.New("OverflowComplex")
		}

		if (dst.IsZero() || c.overwrite) && (c1 != complex128(0) || c.overwriteWithEmptyValue) {
			if c.typeCheck && c.overwrite {
				if dst.Type() != src.Type() {
					return fmt.Errorf("overwrite two different types %s <- %s", dst.Type(), src.Type())
				}
			}

			debugf("%q (%s, %#v) <- (%s, %#v)\n", path, dst.Type(), dst, src.Type(), src)
			dst.SetComplex(c1)
		}

		return nil
	default:
	}

	// Normal map suffices
	if dst.Kind() != src.Kind() {
		return fmt.Errorf("%s can not represents %s", dst.Kind().String(), src.Kind().String())
	}

	dt := dst.Type()
	st := src.Type()
	if !st.AssignableTo(dt) && !st.ConvertibleTo(dt) {
		return fmt.Errorf("%s is not assignable to and convertible to %s", st.String(), dt.String())
	}

	if (dst.IsZero() || c.overwrite) && (!src.IsZero() || c.overwriteWithEmptyValue) {
		if c.typeCheck && c.overwrite {
			if dt != st {
				return fmt.Errorf("overwrite two different types %s <- %s", dt, st)
			}
		}

		debugf("%q (%s, %#v) <- (%s, %#v)\n", path, dt, dst, st, src)
		if st.AssignableTo(dt) {
			dst.Set(src)
		} else {
			dst.Set(src.Convert(dt))
		}
	}
	return nil
}

// DeepMap “deeply map,” the contents of src into dst defined as follows.
// Two values of identical kind are always deeply map if one of the following cases applies.
// Values of distinct kinds can may be deeply map.
//
// Array values are deeply map their corresponding elements.
//
// Struct values are deeply map their corresponding exported fields.
// As a special case, src can have kind Map, keys will be dst fields' names in lower camel case.
//
// Func values deeply map if dst is nil and src is not and both have the same signature; otherwise they not deeply mapped.
//
// Interface values are deeply map they hold concrete values.
//
// Map values deeply map when all of the following are true:
// either they are the same map object or their corresponding keys
// (matched using Go equality) map to deeply map values.
// As a special case, src can have kind Struct, keys will be src fields' names in lower camel case.
//
// Pointer values are deeply map if they are equal using Go's == operator
// or if they point to deeply map values.
//
// Slice values deeply map when all of the following are true:
// either they point to the same initial entry of the same underlying array
// (that is, &x[0] == &y[0]) or their corresponding elements (up to length) deeply mapped.
//
// Other values - numbers, bools, strings, and channels - deeply mapped
// if dst is zero value and src is not, they deeply mapped dst = src using Go's = operator.
//
// Numeric types values deeply map without precision lost and overflow.
//
// String values alse deeply map from a signed or unsigned integer value, slices of bytes,
// and slices of runes.
//
// On the other hand, pointer values are always equal to themselves,
// even if they point at or contain such problematic values,
// because they compare equal using Go's == operator, and that
// is a sufficient condition to be deeply mapped, regardless of content.
// DeepMap has been defined so that the same short-cut applies
// to slices and maps: if x and y are the same slice or the same map,
// they are deeply mapped regardless of content.
//
// As DeepMap traverses the data values it may find a cycle. The
// second and subsequent times that DeepMap compares two pointer
// values that have been mapped before, it treats the values as
// mapped rather than examining the values to which they point.
// This ensures that DeepMap terminates.
func DeepMap(dst, src any, opts ...Option) error {
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

	switch vsrc.Kind() {
	case reflect.Struct:
		switch vdst.Kind() {
		default:
			return errors.New("dst was expected to be a struct or a map")
		case reflect.Struct, reflect.Map:
		}
	case reflect.Map:
		switch vdst.Kind() {
		default:
			return errors.New("dst was expected to be a map or a struct")
		case reflect.Map, reflect.Struct:
		}
	}

	var c Config
	Options(opts).apply(&c)

	return deepValueMap("", vdst, vsrc, make(map[visit]string), &c)
}
