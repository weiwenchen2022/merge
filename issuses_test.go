// Copyright 2013 Dario Castañé. All rights reserved.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package merge_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	. "github.com/weiwenchen2022/merge"

	"github.com/google/go-cmp/cmp"
)

func TestIssue17MergeWithOverwrite(t *testing.T) {
	t.Parallel()

	const doc = `{"timestamp": null, "name": "foo"}`
	var dst map[string]any
	if err := json.Unmarshal([]byte(doc), &dst); err != nil {
		t.Fatal(err)
	}

	test := test{
		dst: dst,
		src: map[string]any{
			"timestamp": nil,
			"name":      "bar",
			"newStuff":  "foo",
		},
		mergeOpts: Options{WithOverwrite()},

		want: map[string]any{
			"timestamp": nil,
			"name":      "bar",
			"newStuff":  "foo",
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue23MergeWithOverwrite(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }

	now := time.Now()
	created := time.Unix(1136214245, 0)

	test := test{
		dst:       &T{now},
		src:       T{created},
		mergeOpts: Options{WithOverwrite()},
		want:      &T{created},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue33Merge(t *testing.T) {
	t.Parallel()

	type T struct {
		A string
		B []int
	}

	tests := []test{
		{
			dst: &T{A: "foo"},
			src: T{"bar", []int{1, 2, 3}},
			// Merge doesn't overwrite an attribute if in destination it doesn't have a zero value.
			// In this case, Str isn't a zero value string.
			want: &T{"foo", []int{1, 2, 3}},
		},
		{
			dst: &T{A: "foo"},
			src: T{"bar", []int{1, 2, 3}},

			// If we want to override, we must use DeepMerge with WithOverwrite.
			mergeOpts: Options{WithOverwrite()},
			want:      &T{"bar", []int{1, 2, 3}},
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestIssue38Merge(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }

	now := time.Now()
	created := time.Unix(1136214245, 0)

	tests := []test{
		{
			dst:  &T{now},
			src:  T{created},
			want: &T{now},
		},
		{
			name: "EmptyStruct",
			dst:  &T{},
			src:  T{created},
			want: &T{created},
		},
		{
			name:      "WithOverwrite",
			dst:       &T{now},
			src:       T{created},
			mergeOpts: Options{WithOverwrite()},
			want:      &T{created},
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestOverwriteZeroSrcTime(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }
	now := time.Now()

	test := test{
		dst:       &T{now},
		src:       T{},
		mergeOpts: Options{WithOverwrite()},

		want: &T{now},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func timeTransformer(overwrite bool) func(*time.Time, time.Time) error {
	return func(dst *time.Time, src time.Time) error {
		if overwrite {
			if !src.IsZero() {
				*dst = src
			}
		} else {
			if dst.IsZero() {
				*dst = src
			}
		}
		return nil
	}
}

func TestOverwriteZeroSrcTimeWithTransformer(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }
	now := time.Now()

	test := test{
		dst:       &T{now},
		src:       T{},
		mergeOpts: Options{WithOverwrite(), WithTransformer(timeTransformer(true))},
		want:      &T{now},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestZeroDstTime(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }
	now := time.Now()

	test := test{
		dst:  &T{},
		src:  T{now},
		want: &T{now},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestZeroDstTimeWithTransformer(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }
	now := time.Now()

	test := test{
		dst:       &T{},
		src:       T{now},
		mergeOpts: Options{WithTransformer(timeTransformer(false))},
		want:      &T{now},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue61MergeNilMap(t *testing.T) {
	t.Parallel()

	type T struct {
		M map[string][]int
	}
	var (
		dst T
		src = T{map[string][]int{"foo": {1, 2, 3}}}
	)
	test := test{
		dst:  &dst,
		src:  src,
		want: &src,
		check: func(t testing.TB, a any) {
			dst := a.(*T)
			if fmt.Sprintf("%p", src.M["foo"]) == fmt.Sprintf("%p", dst.M["foo"]) {
				t.Error("dst and src slice shared underlying array")
			}
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue64MergeSliceWithOverride(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			dst:       New([]string{"bar"}),
			src:       []string{"foo", "bar"},
			mergeOpts: Options{WithOverwrite()},
			want:      New([]string{"foo", "bar"}),
		},

		{
			dst:       New([]string(nil)),
			src:       []string{"foo", "bar"},
			mergeOpts: Options{WithOverwrite()},
			want:      New([]string{"foo", "bar"}),
		},
		{
			dst:       New([]string{}),
			src:       []string{"foo", "bar"},
			mergeOpts: Options{WithOverwrite()},
			want:      New([]string{"foo", "bar"}),
		},

		{
			dst:       New([]string{"foo"}),
			src:       []string(nil),
			mergeOpts: Options{WithOverwrite()},
			want:      New([]string{"foo"}),
		},
		{
			dst:       New([]string{"foo"}),
			src:       []string{},
			mergeOpts: Options{WithOverwrite()},
			want:      New([]string{"foo"}),
		},

		{
			dst:  New([]string(nil)),
			src:  []string{},
			want: New([]string(nil)),
		},
		{
			dst:  New([]string{}),
			src:  []string(nil),
			want: New([]string{}),
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestPrivateSliceWithOverwrite(t *testing.T) {
	t.Parallel()

	type T struct {
		S []string
		s []string
	}

	test := test{
		dst: &T{
			[]string{"foo", "bar"},
			[]string{"a", "b", "c"},
		},
		src: T{
			[]string{"FOO", "BAR"},
			[]string{"A", "B", "C"},
		},
		mergeOpts: Options{WithOverwrite()},

		want: &T{
			[]string{"FOO", "BAR"},
			[]string{"a", "b", "c"},
		},
		cmpOpts: cmp.Options{cmp.AllowUnexported(T{})},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestPrivateSliceWithAppendSlice(t *testing.T) {
	t.Parallel()

	type T struct {
		S []string
		s []string
	}
	test := test{
		dst: &T{
			[]string{"foo", "bar"},
			[]string{"a", "b", "c"},
		},
		src: T{
			[]string{"FOO", "BAR"},
			[]string{"A", "B", "C"},
		},
		mergeOpts: Options{WithAppendSlice()},

		want: &T{
			[]string{"foo", "bar", "FOO", "BAR"},
			[]string{"a", "b", "c"},
		},
		cmpOpts: cmp.Options{cmp.AllowUnexported(T{})},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue83(t *testing.T) {
	t.Parallel()

	test := test{
		dst:       New([]string{"foo", "bar"}),
		src:       []string(nil),
		mergeOpts: Options{WithOverwriteWithEmptyValue()},
		want:      New([]string{"", ""}),
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue89Boolean(t *testing.T) {
	t.Parallel()

	test := test{
		dst:  New(false),
		src:  true,
		want: New(true),
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue89MergeWithEmptyValue(t *testing.T) {
	t.Parallel()

	test := test{
		dst:       map[string]any{"A": 3, "B": "note", "C": true},
		src:       map[string]any{"B": "", "C": false},
		mergeOpts: Options{WithOverwriteWithEmptyValue()},
		want:      map[string]any{"B": "", "C": false},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue90(t *testing.T) {
	t.Parallel()

	type T struct{ M map[string][]int }
	test := test{
		dst:  map[string]T{"foo": {}},
		src:  map[string]T{"foo": {map[string][]int{"foo": {1, 2, 3}}}},
		want: map[string]T{"foo": {map[string][]int{"foo": {1, 2, 3}}}},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue121WithWithOverwrite(t *testing.T) {
	t.Parallel()

	test := test{
		dst: map[string]any{
			"inner": map[string]any{"a": 1, "b": 2},
		},
		src: map[string]any{
			"inner": map[string]any{"a": 3, "c": 4},
		},
		mergeOpts: Options{WithOverwrite()},

		want: map[string]any{
			"inner": map[string]any{"a": 3, "b": 2, "c": 4},
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue123(t *testing.T) {
	t.Parallel()

	test := test{
		dst: map[string]any{
			"a": 1,
			"b": 3,
			"c": 3,
		},
		src: map[string]any{
			"a": nil,
			"b": 4,
			"c": nil,
		},
		mergeOpts: Options{WithOverwrite()},

		want: map[string]any{
			"a": 1,
			"b": 4,
			"c": 3,
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue125MergeWithOverwriteEmptySlice(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			dst:       New([]int{}),
			src:       []int(nil),
			mergeOpts: Options{WithOverwriteEmptySlice()},

			want: New([]int(nil)),
		},
		{
			dst:       New([]int(nil)),
			src:       []int{},
			mergeOpts: Options{WithOverwriteEmptySlice()},

			want: New([]int{}),
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestIssue129Boolean(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			// Standard behavior
			dst:  New(true),
			src:  false,
			want: New(true),
		},
		{
			// Expected behavior
			dst:       New(true),
			src:       false,
			mergeOpts: Options{WithOverwriteWithEmptyValue()},
			want:      New(false),
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestIssue131MergeWithOverwriteWithEmptyValue(t *testing.T) {
	t.Parallel()

	type T struct {
		A *bool
		B string
	}
	test := test{
		dst:       &T{New(true), "foo"},
		src:       T{New(false), "bar"},
		mergeOpts: Options{WithOverwriteWithEmptyValue()},
		want:      &T{New(false), "bar"},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue131MergeWithoutDereferenceWithOverride(t *testing.T) {
	t.Parallel()

	type T struct {
		A *bool
		B string
		C *bool
		D *bool
		E *bool
	}
	var (
		dst = T{New(true), "foo", New(false), nil, New(false)}
		src = T{New(false), "bar", nil, New(false), New(true)}
	)
	test := test{
		dst:       &dst,
		src:       src,
		mergeOpts: Options{WithOverwrite(), WithoutDereference()},

		want: &T{New(false), "bar", New(false), New(false), New(true)},
		check: func(t testing.TB, a any) {
			dst := a.(*T)
			if src.A != dst.A || src.C == dst.C || src.D != dst.D || src.E != dst.E {
				t.Error("pointer values not merged in properly")
			}
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue131MergeWithoutDereference(t *testing.T) {
	t.Parallel()

	type T struct {
		A *bool
		B string
		C *bool
		D *bool
		E *bool
	}
	var (
		dst = T{New(true), "foo", New(false), nil, New(false)}
		src = T{New(false), "bar", nil, New(false), New(true)}
	)
	test := test{
		dst:       &dst,
		src:       src,
		mergeOpts: Options{WithoutDereference()},
		want:      &T{New(true), "foo", New(false), New(false), New(false)},
		check: func(t testing.TB, a any) {
			dst := a.(*T)
			if src.A == dst.A || src.C == dst.C || src.D != dst.D || src.E == dst.E {
				t.Error("pointer valuse not merged in properly")
			}
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestMergeEmbedded(t *testing.T) {
	t.Parallel()

	type embeddedTest struct {
		A string
		B int
	}
	type embeddingTest struct {
		A string
		embeddedTest
	}
	var (
		dst embeddingTest
		src = embeddedTest{"foo", 23}
	)
	test := test{
		dst:  &dst.embeddedTest,
		src:  src,
		want: &src,
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue149(t *testing.T) {
	t.Parallel()

	type T struct{ A string }

	type T1 struct {
		T *T
		B *string
	}
	test := test{
		dst:       &T1{&T{"foo"}, nil},
		src:       &T1{nil, New("bar")},
		mergeOpts: Options{WithOverwriteWithEmptyValue()},

		want: &T1{&T{}, New("bar")},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue174(t *testing.T) {
	t.Parallel()

	type T struct {
		_ int
		A int
	}
	test := test{
		dst:  &T{},
		src:  T{0, 23},
		want: &T{0, 23},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue202(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name: "slice overwrite string",
			dst: map[string]any{
				"foo": 123,
				"bar": "456",
			},
			src: map[string]any{
				"foo": "123",
				"bar": []int{1, 2, 3},
			},
			mergeOpts: Options{WithOverwrite()},

			want: map[string]any{
				"foo": "123",
				"bar": []int{1, 2, 3},
			},
		},
		{
			name: "string overwrite slice",
			dst: map[string]any{
				"foo": 123,
				"bar": []int{1, 2, 3},
			},
			src: map[string]any{
				"foo": "123",
				"bar": "456",
			},
			mergeOpts: Options{WithOverwrite()},

			want: map[string]any{
				"foo": "123",
				"bar": "456",
			},
		},
		{
			name: "map overwrite string",
			dst: map[string]any{
				"foo": 123,
				"bar": "456",
			},
			src: map[string]any{
				"foo": "123",
				"bar": map[string]any{
					"bar": true,
				},
			},
			mergeOpts: Options{WithOverwrite()},

			want: map[string]any{
				"foo": "123",
				"bar": map[string]any{
					"bar": true,
				},
			},
		},
		{
			name: "string overwrite map",
			dst: map[string]any{
				"foo": 123,
				"bar": map[string]any{
					"bar": true,
				},
			},
			src: map[string]any{
				"foo": "123",
				"bar": "456",
			},
			mergeOpts: Options{WithOverwrite()},
			want: map[string]any{
				"foo": "123",
				"bar": "456",
			},
		},
		{
			name: "map overwrite map",
			dst: map[string]any{
				"foo": 123,
				"bar": map[string]any{
					"bar": 456,
				},
			},
			src: map[string]any{
				"foo": "123",
				"bar": map[string]any{
					"bar": "456",
				},
			},
			mergeOpts: Options{WithOverwrite()},

			want: map[string]any{
				"foo": "123",
				"bar": map[string]any{
					"bar": "456",
				},
			},
		},
		{
			name: "map overwrite map with merge",
			dst: map[string]any{
				"foo": 123,
				"bar": map[string]any{
					"a": 1,
					"b": 2,
				},
			},
			src: map[string]any{
				"foo": "123",
				"bar": map[string]any{
					"a": true,
				},
			},
			mergeOpts: Options{WithOverwrite()},

			want: map[string]any{
				"foo": "123",
				"bar": map[string]any{
					"a": true,
					"b": 2,
				},
			},
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestIssue209(t *testing.T) {
	t.Parallel()

	test := test{
		dst:       &[]int{1, 2, 3},
		src:       []int{4, 5},
		mergeOpts: Options{WithAppendSlice()},
		want:      &[]int{1, 2, 3, 4, 5},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestIssue220(t *testing.T) {
	t.Parallel()

	test := test{
		dst:       []any{map[string][]int{"foo": {1, 2, 3}}},
		src:       []any{"bar"},
		mergeOpts: Options{WithoutDereference()},

		want: []any{map[string][]int{"foo": {1, 2, 3}}},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestMergeMapWithOverwrite(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int{"a": 1, "c": 3},
			mergeOpts: Options{WithOverwriteWithEmptyValue()},
			want:      map[string]int{"a": 1, "c": 3},
		},
		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int{"a": 1, "c": 3},
			mergeOpts: Options{WithOverwrite()},
			want:      map[string]int{"a": 1, "b": 2, "c": 3},
		},

		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int{},
			mergeOpts: Options{WithOverwriteWithEmptyValue()},
			want:      map[string]int{},
		},
		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int(nil),
			mergeOpts: Options{WithOverwriteWithEmptyValue()},
			want:      map[string]int{},
		},

		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int{},
			mergeOpts: Options{WithOverwrite()},
			want:      map[string]int{"a": 1, "b": 2},
		},
		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int(nil),
			mergeOpts: Options{WithOverwrite()},
			want:      map[string]int{"a": 1, "b": 2},
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestMergeSliceWithOverrideWithAppendSlice(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			dst:       &[]int{1, 2, 3},
			src:       []int{4, 5},
			mergeOpts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{1, 2, 3, 4, 5},
		},
		{
			dst:       New([]int(nil)),
			src:       []int{4, 5},
			mergeOpts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{4, 5},
		},
		{
			dst:       &[]int{},
			src:       []int{4, 5},
			mergeOpts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{4, 5},
		},
		{

			dst:       &[]int{1, 2, 3},
			src:       []int{},
			mergeOpts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{1, 2, 3},
		},
		{

			dst:       &[]int{1, 2, 3},
			src:       []int(nil),
			mergeOpts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{1, 2, 3},
		},

		{
			dst:       &[]int{},
			src:       []int{},
			mergeOpts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{},
		},
		{
			dst:       New([]int(nil)),
			src:       []int{},
			mergeOpts: Options{WithOverwrite(), WithAppendSlice()},
			want:      New([]int(nil)),
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestMergeMapEmptyString(t *testing.T) {
	t.Parallel()

	type M map[string]any
	test := test{
		dst:  M{"foo": ""},
		src:  M{"foo": "bar"},
		want: M{"foo": "bar"},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestMapInterfaceWithMultipleLayer(t *testing.T) {
	t.Parallel()

	test := test{
		dst: map[string]any{
			"k1": map[string]any{
				"k1.1": "v1",
			},
		},
		src: map[string]any{
			"k1": map[string]any{
				"k1.1": "v2",
				"k1.2": "v3",
			},
		},
		mergeOpts: Options{WithOverwrite()},

		want: map[string]any{
			"k1": map[string]any{
				"k1.1": "v2",
				"k1.2": "v3",
			},
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func Test_deepValueMergeTransformerInvalidDestination(t *testing.T) {
	t.Parallel()

	// this test is intentionally not asserting on anything, it's sole
	// purpose to verify deepValueMerge doesn't panic when a transformer is
	// passed and the destination is invalid.
	f := func(dst *time.Time, src time.Time) error {
		return nil
	}
	t.Run("Merge", func(t *testing.T) {
		t.Cleanup(func() {
			if recover() != nil {
				t.Error("unexpected panicked")
			}
		})
		DeepValueMerge(reflect.Value{}, reflect.ValueOf(time.Now()), WithTransformer(f))
	})

	t.Run("Map", func(t *testing.T) {
		t.Cleanup(func() {
			if recover() != nil {
				t.Error("unexpected panicked")
			}
		})
		DeepValueMap(reflect.Value{}, reflect.ValueOf(time.Now()), WithTransformer(f))
	})
}

func TestMergeWithTransformerZeroValue(t *testing.T) {
	t.Parallel()

	// This test specifically tests that a transformer can be used to
	// prevent overwriting a zero value (in this case a bool). This would fail prior to #211
	test := test{
		dst: New(false),
		src: true,
		mergeOpts: Options{WithTransformer(func(*bool, bool) error {
			return nil
		})},
		want: New(false),
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestV039Issue139(t *testing.T) {
	t.Parallel()

	type inner struct{ A int }
	type outer struct {
		inner
		B int
	}
	test := test{
		dst:       &outer{inner{1}, 2},
		src:       outer{inner{10}, 20},
		mergeOpts: Options{WithOverwrite()},

		want:    &outer{inner{10}, 20},
		cmpOpts: cmp.Options{cmp.AllowUnexported(outer{})},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestV039Issue146(t *testing.T) {
	t.Parallel()

	type Bar struct{ A, B *string }
	type Foo struct {
		A string
		B map[string]Bar
	}
	var s1, s2 = "asd", "sdf"
	dst := Foo{"bar", map[string]Bar{"foo": {&s1, nil}}}
	src := Foo{"foo", map[string]Bar{"foo": {nil, &s2}}}
	test := test{
		dst:       &dst,
		src:       src,
		mergeOpts: Options{WithOverwrite()},

		want: &Foo{"foo", map[string]Bar{"foo": {&s1, &s2}}},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}
