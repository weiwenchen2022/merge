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

	test := test{
		dst:  map[string]any{"K1": "V1", "K3": "V3"},
		src:  map[string]any{"K1": "v1", "K2": "v2"},
		want: map[string]any{"K1": "V1", "K2": "v2", "K3": "V3"},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

type T struct {
	A int
}

func TestSimpleStruct(t *testing.T) {
	t.Parallel()

	test := test{dst: &T{}, src: T{42}, want: &T{42}}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

type T2 struct {
	A string
	T T
	c int
}

func TestComplexStruct(t *testing.T) {
	t.Parallel()

	test := test{
		dst: &T2{A: "foo"},
		src: T2{"bar", T{42}, 1},

		want:    &T2{"foo", T{42}, 0},
		cmpOpts: cmp.Options{cmp.AllowUnexported(T2{})},
	}
	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

}

func TestComplexStructWithOverwrite(t *testing.T) {
	t.Parallel()

	test := test{
		dst:       &T2{"do-not-overwrite-with-empty-value", T{23}, 1},
		src:       T2{T: T{42}, c: 2},
		mergeOpts: Options{WithOverwrite()},
		want:      &T2{"do-not-overwrite-with-empty-value", T{42}, 1},
		cmpOpts:   cmp.Options{cmp.AllowUnexported(T2{})},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

type PT struct {
	T *T
}

func TestPointerStruct(t *testing.T) {
	t.Parallel()

	test := test{
		dst:  &PT{},
		src:  PT{&T{19}},
		want: &PT{&T{19}},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
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

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func sliceTests(t *testing.T, dst, src []int, opts Options, want []int) []test {
	type S struct{ S []int }

	tests := []test{
		{
			dst:       New(dst),
			src:       src,
			mergeOpts: opts,
			want:      New(want),
		},
		{
			dst:       &S{dst},
			src:       S{src},
			mergeOpts: opts,
			want:      &S{want},
		},
		{
			dst:       map[string][]int{"S": dst},
			src:       map[string][]int{"S": src},
			mergeOpts: opts,
			want:      map[string][]int{"S": want},
		},
	}
	if dst == nil {
		// test case with missing dst key
		tests = append(tests, test{
			dst:       make(map[string][]int),
			src:       map[string][]int{"S": src},
			mergeOpts: opts,
			want:      map[string][]int{"S": src},
		})
	}
	if src == nil {
		// test case with missing src key
		tests = append(tests, test{
			dst:       map[string][]int{"S": dst},
			src:       map[string][]int(nil),
			mergeOpts: opts,
			want:      map[string][]int{"S": dst},
		})
	}

	return tests
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

	t.Run("Merge", func(t *testing.T) {
		for i, tt := range tests {
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				tests := sliceTests(t, tt.dst, tt.src, tt.opts, tt.want)
				testDeepMerge(t, tests...)
			})
		}
	})

	t.Run("Map", func(t *testing.T) {
		for i, tt := range tests {
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				tests := sliceTests(t, tt.dst, tt.src, tt.opts, tt.want)
				testDeepMap(t, tests...)
			})
		}
	})
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

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestEmptyToNonEmptyMap(t *testing.T) {
	t.Parallel()

	test := test{
		dst:  map[string]int{"foo": 23, "bar": 42},
		src:  map[string]int(nil),
		want: map[string]int{"foo": 23, "bar": 42},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestMapWithOverwrite(t *testing.T) {
	t.Parallel()

	test := test{
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
		mergeOpts: Options{WithOverwrite()},

		want: map[string]T{
			"a": {16},
			"b": {42},
			"c": {12},
			"d": {61},
			"e": {14},
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestMapWithEmbeddedStructPointer(t *testing.T) {
	t.Parallel()

	test := test{
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
		mergeOpts: Options{WithOverwrite()},

		want: map[string]*T{
			"a": {16},
			"b": {42},
			"c": {12},
			"d": {61},
			"e": {14},
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
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
			mergeOpts: Options{WithOverwrite()},

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

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestMap(t *testing.T) {
	t.Parallel()

	test := test{
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
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestMapWithNilPointer(t *testing.T) {
	t.Parallel()

	test := test{
		dst:  map[string]*int{"a": nil, "b": nil},
		src:  map[string]*int{"b": nil, "c": nil},
		want: map[string]*int{"a": nil, "b": nil, "c": nil},
	}

	t.Run("merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestTwoPointerValues(t *testing.T) {
	t.Parallel()

	var dst *int
	src := New(42)
	test := test{
		dst:  &dst,
		src:  src,
		want: &src,
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestUnexportedProperty(t *testing.T) {
	t.Parallel()

	type T struct {
		a string
	}
	test := test{
		dst: map[string]T{"key": {"hello"}},
		src: map[string]T{"key": {"hi"}},
	}

	t.Run("Merge", func(t *testing.T) {
		t.Cleanup(func() {
			if recover() != nil {
				t.Error("unexpected panic")
			}
		})
		testDeepMerge(t, test)
	})

	t.Run("Map", func(t *testing.T) {
		t.Cleanup(func() {
			if recover() != nil {
				t.Error("unexpected panic")
			}
		})
		testDeepMap(t, test)
	})
}

func TestBooleanPointer(t *testing.T) {
	t.Parallel()

	type T struct{ B *bool }

	test := test{
		dst:  &T{},
		src:  T{New(true)},
		want: &T{New(true)},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestMergeMapWithInnerSliceOfDifferentType(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name:      "With overwrite and append slice",
			dst:       map[string]any{"foo": []int{1, 2}},
			src:       map[string]any{"foo": []string{"a", "b"}},
			mergeOpts: Options{WithOverwrite(), WithAppendSlice()},
			wantErr:   true,
		},
		{
			name:      "With overwrite and type check",
			dst:       map[string]any{"foo": []int{1, 2}},
			src:       map[string]any{"foo": []string{"a", "b"}},
			mergeOpts: Options{WithOverwrite(), WithTypeCheck()},
			wantErr:   true,
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestMergeDifferentSlicesIsNotSupported(t *testing.T) {
	t.Parallel()

	test := test{
		dst:       New([]int{1, 2}),
		src:       []string{"a", "b"},
		mergeOpts: Options{WithOverwrite(), WithAppendSlice()},
		wantErr:   true,
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}
