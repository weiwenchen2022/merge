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

	testDeepMerge(t, test{
		dst: dst,
		src: map[string]any{
			"timestamp": nil,
			"name":      "bar",
			"newStuff":  "foo",
		},
		mergeopts: Options{WithOverwrite()},

		want: map[string]any{
			"timestamp": nil,
			"name":      "bar",
			"newStuff":  "foo",
		},
	})
}

func TestIssue23MergeWithOverwrite(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }

	now := time.Now()
	created := time.Unix(1136214245, 0)

	testDeepMerge(t, test{
		dst:       &T{now},
		src:       T{created},
		mergeopts: Options{WithOverwrite()},
		want:      &T{created},
	})
}

func TestIssue33Merge(t *testing.T) {
	t.Parallel()

	type T struct {
		A string
		B []int
	}

	testDeepMerge(t, []test{
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
			mergeopts: Options{WithOverwrite()},
			want:      &T{"bar", []int{1, 2, 3}},
		},
	}...)
}

func TestIssue38Merge(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }

	now := time.Now()
	created := time.Unix(1136214245, 0)

	testDeepMerge(t, []test{
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
			mergeopts: Options{WithOverwrite()},
			want:      &T{created},
		},
	}...)
}

func TestOverwriteZeroSrcTime(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }
	now := time.Now()

	testDeepMerge(t, test{
		dst:       &T{now},
		src:       T{},
		mergeopts: Options{WithOverwrite()},

		want: &T{now},
	})
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

	testDeepMerge(t, test{
		dst:       &T{now},
		src:       T{},
		mergeopts: Options{WithOverwrite(), WithTransformer(timeTransformer(true))},
		want:      &T{now},
	})
}

func TestZeroDstTime(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }
	now := time.Now()

	testDeepMerge(t, test{
		dst:  &T{},
		src:  T{now},
		want: &T{now},
	})
}

func TestZeroDstTimeWithTransformer(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }
	now := time.Now()

	testDeepMerge(t, test{
		dst:       &T{},
		src:       T{now},
		mergeopts: Options{WithTransformer(timeTransformer(false))},
		want:      &T{now},
	})
}

func TestIssue61MergeNilMap(t *testing.T) {
	t.Parallel()

	type T struct {
		M map[string][]int
	}

	var (
		dst = T{}
		src = T{map[string][]int{"foo": {1, 2, 3}}}
	)

	testDeepMerge(t, test{
		dst:  &dst,
		src:  src,
		want: &src,
	})
	if fmt.Sprintf("%p", src.M["foo"]) == fmt.Sprintf("%p", dst.M["foo"]) {
		t.Error("dst and src slice shared underlying array")
	}
}

func TestIssue64MergeSliceWithOverride(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, []test{
		{
			dst:       New([]string{"bar"}),
			src:       []string{"foo", "bar"},
			mergeopts: Options{WithOverwrite()},
			want:      New([]string{"foo", "bar"}),
		},

		{
			dst:       New([]string(nil)),
			src:       []string{"foo", "bar"},
			mergeopts: Options{WithOverwrite()},
			want:      New([]string{"foo", "bar"}),
		},
		{
			dst:       New([]string{}),
			src:       []string{"foo", "bar"},
			mergeopts: Options{WithOverwrite()},
			want:      New([]string{"foo", "bar"}),
		},

		{
			dst:       New([]string{"foo"}),
			src:       []string(nil),
			mergeopts: Options{WithOverwrite()},
			want:      New([]string{"foo"}),
		},
		{
			dst:       New([]string{"foo"}),
			src:       []string{},
			mergeopts: Options{WithOverwrite()},
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
	}...)
}

func TestPrivateSliceWithOverwrite(t *testing.T) {
	t.Parallel()

	type T struct {
		S []string
		s []string
	}

	testDeepMerge(t, test{
		dst: &T{
			[]string{"foo", "bar"},
			[]string{"a", "b", "c"},
		},
		src: T{
			[]string{"FOO", "BAR"},
			[]string{"A", "B", "C"},
		},
		mergeopts: Options{WithOverwrite()},

		want: &T{
			[]string{"FOO", "BAR"},
			[]string{"a", "b", "c"},
		},
		cmpopts: cmp.Options{cmp.AllowUnexported(T{})},
	})
}

func TestPrivateSliceWithAppendSlice(t *testing.T) {
	t.Parallel()

	type T struct {
		S []string
		s []string
	}

	testDeepMerge(t, test{
		dst: &T{
			[]string{"foo", "bar"},
			[]string{"a", "b", "c"},
		},
		src: T{
			[]string{"FOO", "BAR"},
			[]string{"A", "B", "C"},
		},
		mergeopts: Options{WithAppendSlice()},

		want: &T{
			[]string{"foo", "bar", "FOO", "BAR"},
			[]string{"a", "b", "c"},
		},
		cmpopts: cmp.Options{cmp.AllowUnexported(T{})},
	})
}

func TestIssue83(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:       New([]string{"foo", "bar"}),
		src:       []string(nil),
		mergeopts: Options{WithOverwriteWithEmptyValue()},
		want:      New([]string{"", ""}),
	})
}

func TestIssue89Boolean(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:  New(false),
		src:  true,
		want: New(true),
	})
}

func TestIssue89MergeWithEmptyValue(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:       map[string]any{"A": 3, "B": "note", "C": true},
		src:       map[string]any{"B": "", "C": false},
		mergeopts: Options{WithOverwriteWithEmptyValue()},
		want:      map[string]any{"B": "", "C": false},
	})
}

func TestIssue90(t *testing.T) {
	t.Parallel()

	type T struct{ M map[string][]int }

	testDeepMerge(t, test{
		dst:  map[string]T{"foo": {}},
		src:  map[string]T{"foo": {map[string][]int{"foo": {1, 2, 3}}}},
		want: map[string]T{"foo": {map[string][]int{"foo": {1, 2, 3}}}},
	})
}

func TestIssue121WithWithOverwrite(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst: map[string]any{
			"inner": map[string]any{"a": 1, "b": 2},
		},
		src: map[string]any{
			"inner": map[string]any{"a": 3, "c": 4},
		},
		mergeopts: Options{WithOverwrite()},

		want: map[string]any{
			"inner": map[string]any{"a": 3, "b": 2, "c": 4},
		},
	})
}

func TestIssue123(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
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
		mergeopts: Options{WithOverwrite()},

		want: map[string]any{
			"a": 1,
			"b": 4,
			"c": 3,
		},
	})
}

func TestIssue125MergeWithOverwriteEmptySlice(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, []test{
		{
			dst:       New([]int{}),
			src:       []int(nil),
			mergeopts: Options{WithOverwriteEmptySlice()},

			want: New([]int(nil)),
		},
		{
			dst:       New([]int(nil)),
			src:       []int{},
			mergeopts: Options{WithOverwriteEmptySlice()},

			want: New([]int{}),
		},
	}...)
}

func TestIssue129Boolean(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, []test{
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
			mergeopts: Options{WithOverwriteWithEmptyValue()},
			want:      New(false),
		},
	}...)
}

func TestIssue131MergeWithOverwriteWithEmptyValue(t *testing.T) {
	t.Parallel()

	type T struct {
		A *bool
		B string
	}

	testDeepMerge(t, test{
		dst:       &T{New(true), "foo"},
		src:       T{New(false), "bar"},
		mergeopts: Options{WithOverwriteWithEmptyValue()},
		want:      &T{New(false), "bar"},
	})
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

	testDeepMerge(t, test{
		dst:       &dst,
		src:       src,
		mergeopts: Options{WithOverwrite(), WithoutDereference()},

		want: &T{New(false), "bar", New(false), New(false), New(true)},
	})
	if src.A != dst.A || src.C == dst.C || src.D != dst.D || src.E != dst.E {
		t.Error("pointer values not merged in properly")
	}
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

	testDeepMerge(t, test{
		dst:       &dst,
		src:       src,
		mergeopts: Options{WithoutDereference()},
		want:      &T{New(true), "foo", New(false), New(false), New(false)},
	})
	if src.A == dst.A || src.C == dst.C || src.D != dst.D || src.E == dst.E {
		t.Error("pointer valuse not merged in properly")
	}
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
	testDeepMerge(t, test{
		dst:  &dst.embeddedTest,
		src:  src,
		want: &src,
	})
}

func TestIssue149(t *testing.T) {
	t.Parallel()

	type T struct{ A string }

	type T1 struct {
		T *T
		B *string
	}

	testDeepMerge(t, test{
		dst:       &T1{&T{"foo"}, nil},
		src:       &T1{nil, New("bar")},
		mergeopts: Options{WithOverwriteWithEmptyValue()},

		want: &T1{&T{}, New("bar")},
	})
}

func TestIssue174(t *testing.T) {
	t.Parallel()

	type T struct {
		_ int
		A int
	}

	testDeepMerge(t, test{
		dst:  &T{},
		src:  T{0, 23},
		want: &T{0, 23},
	})
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
			mergeopts: Options{WithOverwrite()},

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
			mergeopts: Options{WithOverwrite()},

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
			mergeopts: Options{WithOverwrite()},

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
			mergeopts: Options{WithOverwrite()},
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
			mergeopts: Options{WithOverwrite()},

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
			mergeopts: Options{WithOverwrite()},

			want: map[string]any{
				"foo": "123",
				"bar": map[string]any{
					"a": true,
					"b": 2,
				},
			},
		},
	}
	testDeepMerge(t, tests...)
}

func TestIssue209(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:       &[]int{1, 2, 3},
		src:       []int{4, 5},
		mergeopts: Options{WithAppendSlice()},
		want:      &[]int{1, 2, 3, 4, 5},
	})
}

func TestIssue220(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:       []any{map[string][]int{"foo": {1, 2, 3}}},
		src:       []any{"bar"},
		mergeopts: Options{WithoutDereference()},

		want: []any{map[string][]int{"foo": {1, 2, 3}}},
	})
}

func TestMergeMapWithOverwrite(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int{"a": 1, "c": 3},
			mergeopts: Options{WithOverwriteWithEmptyValue()},
			want:      map[string]int{"a": 1, "c": 3},
		},
		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int{"a": 1, "c": 3},
			mergeopts: Options{WithOverwrite()},
			want:      map[string]int{"a": 1, "b": 2, "c": 3},
		},

		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int{},
			mergeopts: Options{WithOverwriteWithEmptyValue()},
			want:      map[string]int{},
		},
		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int(nil),
			mergeopts: Options{WithOverwriteWithEmptyValue()},
			want:      map[string]int{},
		},

		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int{},
			mergeopts: Options{WithOverwrite()},
			want:      map[string]int{"a": 1, "b": 2},
		},
		{
			dst:       map[string]int{"a": 1, "b": 2},
			src:       map[string]int(nil),
			mergeopts: Options{WithOverwrite()},
			want:      map[string]int{"a": 1, "b": 2},
		},
	}
	testDeepMerge(t, tests...)
}

func TestMergeSliceWithOverrideWithAppendSlice(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			dst:       &[]int{1, 2, 3},
			src:       []int{4, 5},
			mergeopts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{1, 2, 3, 4, 5},
		},
		{
			dst:       New([]int(nil)),
			src:       []int{4, 5},
			mergeopts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{4, 5},
		},
		{
			dst:       &[]int{},
			src:       []int{4, 5},
			mergeopts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{4, 5},
		},
		{

			dst:       &[]int{1, 2, 3},
			src:       []int{},
			mergeopts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{1, 2, 3},
		},
		{

			dst:       &[]int{1, 2, 3},
			src:       []int(nil),
			mergeopts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{1, 2, 3},
		},

		{
			dst:       &[]int{},
			src:       []int{},
			mergeopts: Options{WithOverwrite(), WithAppendSlice()},
			want:      &[]int{},
		},
		{
			dst:       New([]int(nil)),
			src:       []int{},
			mergeopts: Options{WithOverwrite(), WithAppendSlice()},
			want:      New([]int(nil)),
		},
	}
	testDeepMerge(t, tests...)
}

func TestMergeMapEmptyString(t *testing.T) {
	t.Parallel()

	type M map[string]any

	testDeepMerge(t, test{
		dst:  M{"foo": ""},
		src:  M{"foo": "bar"},
		want: M{"foo": "bar"},
	})
}

func TestMapInterfaceWithMultipleLayer(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
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
		mergeopts: Options{WithOverwrite()},

		want: map[string]any{
			"k1": map[string]any{
				"k1.1": "v2",
				"k1.2": "v3",
			},
		},
	})
}

func Test_deepValueMergeTransformerInvalidDestination(t *testing.T) {
	t.Parallel()

	DeepValueMerge(reflect.Value{}, reflect.ValueOf(time.Now()), WithTransformer(func(dst *time.Time, src time.Time) error {
		return nil
	}))
	// this test is intentionally not asserting on anything, it's sole
	// purpose to verify deepValueMerge doesn't panic when a transformer is
	// passed and the destination is invalid.
}

func TestMergeWithTransformerZeroValue(t *testing.T) {
	t.Parallel()

	// This test specifically tests that a transformer can be used to
	// prevent overwriting a zero value (in this case a bool). This would fail prior to #211
	testDeepMerge(t, test{
		dst: New(false),
		src: true,
		mergeopts: Options{WithTransformer(func(*bool, bool) error {
			return nil
		})},
		want: New(false),
	})
}

func TestV039Issue139(t *testing.T) {
	t.Parallel()

	type inner struct{ A int }
	type outer struct {
		inner
		B int
	}

	testDeepMerge(t, test{
		dst:       &outer{inner{1}, 2},
		src:       outer{inner{10}, 20},
		mergeopts: Options{WithOverwrite()},

		want:    &outer{inner{10}, 20},
		cmpopts: cmp.Options{cmp.AllowUnexported(outer{})},
	})
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

	testDeepMerge(t, test{
		dst:       &dst,
		src:       src,
		mergeopts: Options{WithOverwrite()},

		want: &Foo{"foo", map[string]Bar{"foo": {&s1, &s2}}},
	})
}
