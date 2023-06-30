// Copyright 2013 Dario Castañé. All rights reserved.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package merge_test

import (
	"fmt"
	"testing"

	. "github.com/weiwenchen2022/merge"

	"github.com/google/go-cmp/cmp"
)

func TestKV(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:  map[string]any{"K1": "V1", "K3": "V3"},
		src:  map[string]any{"K1": "v1", "K2": "v2"},
		want: map[string]any{"K1": "V1", "K2": "v2", "K3": "V3"},
	})
}

type T struct {
	A int
}

func TestSimpleStruct(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{dst: &T{}, src: T{42}, want: &T{42}})
}

type T2 struct {
	A string
	T T
	c int
}

func TestComplexStruct(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst: &T2{A: "foo"},
		src: T2{"bar", T{42}, 1},

		want:    &T2{"foo", T{42}, 0},
		cmpopts: cmp.Options{cmp.AllowUnexported(T2{})},
	})
}

func TestComplexStructWithOverwrite(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:       &T2{"do-not-overwrite-with-empty-value", T{23}, 1},
		src:       T2{T: T{42}, c: 2},
		mergeopts: Options{WithOverwrite()},
		want:      &T2{"do-not-overwrite-with-empty-value", T{42}, 1},
		cmpopts:   cmp.Options{cmp.AllowUnexported(T2{})},
	})
}

type PT struct {
	T *T
}

func TestPointerStruct(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:  &PT{},
		src:  PT{&T{19}},
		want: &PT{&T{19}},
	})
}

func TestEmbeddedStruct(t *testing.T) {
	t.Parallel()

	type Embedded struct{ A string }
	type Embedding struct{ Embedded }

	tests := []test{
		{
			dst:  &Embedding{},
			src:  Embedding{Embedded{"foo"}},
			want: &Embedding{Embedded{"foo"}},
		},
		{
			dst:  &Embedding{Embedded{"foo"}},
			src:  Embedding{},
			want: &Embedding{Embedded{"foo"}},
		},
		{
			dst:  &Embedding{Embedded{"foo"}},
			src:  Embedding{Embedded{"bar"}},
			want: &Embedding{Embedded{"foo"}},
		},
	}
	testDeepMerge(t, tests...)
}

func testSlice(t *testing.T, dst, src []int, opts Options, want []int) {
	type S struct{ S []int }

	tests := []test{
		{
			dst:       New(dst),
			src:       src,
			mergeopts: opts,
			want:      New(want),
		},
		{
			dst:       &S{dst},
			src:       S{src},
			mergeopts: opts,
			want:      &S{want},
		},
		{
			dst:       map[string][]int{"S": dst},
			src:       map[string][]int{"S": src},
			mergeopts: opts,
			want:      map[string][]int{"S": want},
		},
	}
	if dst == nil {
		// test case with missing dst key
		tests = append(tests, test{
			dst:       make(map[string][]int),
			src:       map[string][]int{"S": src},
			mergeopts: opts,
			want:      map[string][]int{"S": src},
		})
	}
	if src == nil {
		// test case with missing src key
		tests = append(tests, test{
			dst:       map[string][]int{"S": dst},
			src:       map[string][]int(nil),
			mergeopts: opts,
			want:      map[string][]int{"S": dst},
		})
	}

	testDeepMerge(t, tests...)
}

func TestSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dst, src []int
		opts     Options
		want     []int
	}{
		{src: []int{1, 2, 3}, want: []int{1, 2, 3}},
		{dst: []int{}, src: []int{1, 2, 3}, want: []int{1, 2, 3}},

		{dst: []int{1}, src: []int{1, 2, 3}, want: []int{1, 2, 3}},

		{dst: []int{1}, src: []int{}, want: []int{1}},
		{dst: []int{1}, want: []int{1}},

		{src: []int{1, 2, 3}, opts: Options{WithAppendSlice()}, want: []int{1, 2, 3}},
		{dst: []int{}, src: []int{1, 2, 3}, opts: Options{WithAppendSlice()}, want: []int{1, 2, 3}},

		{dst: []int{1}, src: []int{2, 3}, opts: Options{WithAppendSlice()}, want: []int{1, 2, 3}},
		{dst: []int{1}, src: []int{2, 3}, opts: Options{WithOverwrite(), WithAppendSlice()}, want: []int{1, 2, 3}},

		{dst: []int{1}, src: []int{}, opts: Options{WithAppendSlice()}, want: []int{1}},
		{dst: []int{1}, opts: Options{WithAppendSlice()}, want: []int{1}},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			testSlice(t, tt.dst, tt.src, tt.opts, tt.want)
		})
	}
}

func TestEmptyMap(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			dst:  New(map[string]int(nil)),
			src:  make(map[string]int),
			want: New(map[string]int(nil)),
		},
		{
			dst:  New(map[string]int{}),
			src:  map[string]int(nil),
			want: New(map[string]int{}),
		},
	}
	testDeepMerge(t, tests...)
}

func TestEmptyToNonEmptyMap(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:  map[string]int{"foo": 23, "bar": 42},
		src:  map[string]int(nil),
		want: map[string]int{"foo": 23, "bar": 42},
	})
}

func TestMapWithOverwrite(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst: map[string]T{
			"a": {},   // overwritten by 16
			"b": {42}, // not overwritten by empty value
			"c": {13}, // overwritten by 12
			"d": {61},
		},
		src: map[string]T{
			"a": {16},
			"b": {},
			"c": {12},
			"e": {14},
		},
		mergeopts: Options{WithOverwrite()},

		want: map[string]T{
			"a": {16},
			"b": {42},
			"c": {12},
			"d": {61},
			"e": {14},
		},
	})
}

func TestMapWithEmbeddedStructPointer(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst: map[string]*T{
			"a": {},   // overwritten by 16
			"b": {42}, // not overwritten by empty value
			"c": {13}, // overwritten by 12
			"d": {61},
		},
		src: map[string]*T{
			"a": {16},
			"b": nil,
			"c": {12},
			"e": {14},
		},
		mergeopts: Options{WithOverwrite()},

		want: map[string]*T{
			"a": {16},
			"b": {42},
			"c": {12},
			"d": {61},
			"e": {14},
		},
	})
}

func TestMergeUsingStructAndMap(t *testing.T) {
	t.Parallel()

	type T struct {
		A int
		B string
	}
	type T2 struct {
		A string
		B int
	}
	type T3 struct {
		A  string
		T  *T
		T2 *T2
	}
	type T4 struct {
		A  string
		B  string
		T3 *T3
	}

	tests := []test{
		{
			name: "Should overwrite values in target for non-nil values in source",
			dst: &T4{
				A: "foo",
				T3: &T3{
					A:  "foo",
					T:  &T{23, "foo"},
					T2: &T2{"foo", 0},
				},
			},
			src: &T4{
				B: "bar",
				T3: &T3{
					T2: &T2{"bar", 42},
				},
			},
			mergeopts: Options{WithOverwrite()},

			want: &T4{
				A: "foo",
				B: "bar",
				T3: &T3{
					A:  "foo",
					T:  &T{23, "foo"},
					T2: &T2{"bar", 42},
				},
			},
		},
		{
			name: "Should not overwrite values in target for non-nil values in source",
			dst: &T4{
				A: "foo",
				T3: &T3{
					A:  "foo",
					T:  &T{23, "foo"},
					T2: &T2{"foo", 0},
				},
			},
			src: &T4{
				B: "bar",
				T3: &T3{
					T2: &T2{"bar", 42},
				},
			},
			want: &T4{
				A: "foo",
				B: "bar",
				T3: &T3{
					A:  "foo",
					T:  &T{23, "foo"},
					T2: &T2{"foo", 42},
				},
			},
		},
	}
	testDeepMerge(t, tests...)
}

func TestMap(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst: map[string]int{
			"a": 0,
			"b": 42,
			"c": 13,
			"d": 61,
		},
		src: map[string]int{
			"a": 16,
			"b": 0,
			"c": 12,
			"e": 14,
		},
		want: map[string]int{
			"a": 16,
			"b": 42,
			"c": 13,
			"d": 61,
			"e": 14,
		},
	})
}

func TestMapWithNilPointer(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:  map[string]*int{"a": nil, "b": nil},
		src:  map[string]*int{"b": nil, "c": nil},
		want: map[string]*int{"a": nil, "b": nil, "c": nil},
	})
}

func TestTwoPointerValues(t *testing.T) {
	t.Parallel()

	var dst *int
	src := New(42)
	testDeepMerge(t, test{
		dst:  &dst,
		src:  src,
		want: &src,
	})
}

func TestUnexportedProperty(t *testing.T) {
	t.Parallel()

	type T struct {
		a string
	}

	t.Cleanup(func() {
		if recover() != nil {
			t.Error("unexpected panic")
		}
	})
	testDeepMerge(t, test{
		dst: map[string]T{"key": {"hello"}},
		src: map[string]T{"key": {"hi"}},
	})
}

func TestBooleanPointer(t *testing.T) {
	t.Parallel()

	type T struct{ B *bool }

	testDeepMerge(t, test{
		dst:  &T{},
		src:  T{New(true)},
		want: &T{New(true)},
	})
}

func TestMergeMapWithInnerSliceOfDifferentType(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name:      "With overwrite and append slice",
			dst:       map[string]any{"foo": []int{1, 2}},
			src:       map[string]any{"foo": []string{"a", "b"}},
			mergeopts: Options{WithOverwrite(), WithAppendSlice()},
			wantErr:   true,
		},
		{
			name:      "With overwrite and type check",
			dst:       map[string]any{"foo": []int{1, 2}},
			src:       map[string]any{"foo": []string{"a", "b"}},
			mergeopts: Options{WithOverwrite(), WithTypeCheck()},
			wantErr:   true,
		},
	}
	testDeepMerge(t, tests...)
}

func TestMergeDifferentSlicesIsNotSupported(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:       New([]int{1, 2}),
		src:       []string{"a", "b"},
		mergeopts: Options{WithOverwrite(), WithAppendSlice()},
		wantErr:   true,
	})
}
