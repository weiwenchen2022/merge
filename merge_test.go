// Copyright 2013 Dario Castañé. All rights reserved.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package merge_test

import (
	"testing"

	. "github.com/weiwenchen2022/merge"

	"github.com/google/go-cmp/cmp"
)

func TestMergeWithTransformerNilStruct(t *testing.T) {
	t.Parallel()

	type T struct {
		a int
		m map[string]int
	}
	type T2 struct {
		a string
		T *T
	}

	testDeepMerge(t, test{
		dst: &T2{a: "foo"},
		src: T2{T: &T{23, map[string]int{"foo": 23}}},
		mergeopts: Options{WithOverwrite(), WithTransformer(func(dst **T, src *T) error {
			*dst = New(*src)
			t.Log((*dst).a)
			t.Log(*src)
			return nil
		})},
		want:    &T2{"foo", &T{23, map[string]int{"foo": 23}}},
		cmpopts: cmp.Options{cmp.AllowUnexported(T2{}, T{})},
	})
}

func TestMergeNonPointer(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:     T{},
		src:     T{42},
		wantErr: true,
	})
}

func TestMapNonPointer(t *testing.T) {
	t.Parallel()

	testDeepMerge(t, test{
		dst:     map[string]T(nil),
		src:     map[string]T{"foo": {42}},
		wantErr: true,
	})
}
