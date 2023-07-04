package merge_test

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/weiwenchen2022/merge"

	"github.com/google/go-cmp/cmp"
)

func testDeepMap(t *testing.T, tests ...test) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			dst := makeDst(t, tt.dst)
			if err := DeepMap(dst, tt.src, tt.mergeOpts...); (err != nil) != tt.wantErr {
				if err == nil {
					t.Fatal("want error got nil")
				} else {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Log(err)
			}

			if tt.wantErr {
				return
			}

			if tt.want != nil {
				if !cmp.Equal(tt.want, dst, tt.cmpOpts...) {
					t.Error(cmp.Diff(tt.want, dst, tt.cmpOpts))
				}
			}

			if tt.check != nil {
				tt.check(t, dst)
			}
		})
	}
}

func TestNumericTypes(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name: "float64 to float32",
			dst:  New(float32(0)),
			src:  2.718281828,
			want: New(float32(2.718281828)),
		},
		{
			name: "float64 to float32",
			dst:  New(float32(0)),
			src:  0.49999999,
			want: New(float32(0.49999999)),
		},
		{
			name:    "float64 to int failed",
			dst:     New(int(0)),
			src:     1.2,
			wantErr: true,
		},
		{
			name:    "int to uint8 failed",
			dst:     New(uint8(0)),
			src:     -1,
			wantErr: true,
		},
		{
			name: "int to complex128",
			dst:  New(complex128(0)),
			src:  1,
			want: New(complex128(1)),
		},
		{
			name: "complex128 to int",
			dst:  New(int(0)),
			src:  complex(1, 0),
			want: New(int(1)),
		},
	}

	testDeepMap(t, tests...)
}

func TestIntToString(t *testing.T) {
	t.Parallel()

	type MyString string
	tests := []test{
		{
			dst:  New(MyString("")),
			src:  "foo" + "bar",
			want: New(MyString("foobar")),
		},
		{
			dst:  New(MyString("")),
			src:  'x',
			want: New(MyString("x")),
		},
		{
			dst:  New(MyString("")),
			src:  0x266c,
			want: New(MyString("♬")),
		},
		{
			dst:  New(MyString("")),
			src:  '\u00E4',
			want: New(MyString("ä")),
		},
	}

	testDeepMap(t, tests...)
}

func TestBytesToString(t *testing.T) {
	t.Parallel()

	type (
		MyString string
		MyByte   byte
		MyBytes  []MyByte
	)
	tests := []test{
		{
			dst:  New(MyString("")),
			src:  []byte{'h', 'e', 'l', 'l', '\xc3', '\xb8'},
			want: New(MyString("hellø")),
		},
		{
			dst:  New(MyString("")),
			src:  []MyByte{'h', 'e', 'l', 'l', '\xc3', '\xb8'},
			want: New(MyString("hellø")),
		},
		{
			dst:  New(MyString("")),
			src:  MyBytes{'h', 'e', 'l', 'l', '\xc3', '\xb8'},
			want: New(MyString("hellø")),
		},

		// zero values
		{
			dst:  New(MyString("")),
			src:  []byte{},
			want: New(MyString("")),
		},
		{
			dst:  New(MyString("")),
			src:  []byte(nil),
			want: New(MyString("")),
		},
		{
			dst:  New(MyString("")),
			src:  []MyByte{},
			want: New(MyString("")),
		},
		{
			dst:  New(MyString("")),
			src:  []MyByte(nil),
			want: New(MyString("")),
		},
		{
			dst:  New(MyString("")),
			src:  MyBytes{},
			want: New(MyString("")),
		},
		{
			dst:  New(MyString("")),
			src:  MyBytes(nil),
			want: New(MyString("")),
		},
	}

	testDeepMap(t, tests...)
}

func TestRunesToString(t *testing.T) {
	t.Parallel()

	type (
		MyString string
		MyRune   rune
		MyRunes  []MyRune
	)
	tests := []test{
		{
			dst:  New(MyString("")),
			src:  []rune{0x266b, 0x266c},
			want: New(MyString("♫♬")),
		},
		{
			dst:  New(MyString("")),
			src:  []MyRune{0x266b, 0x266c},
			want: New(MyString("♫♬")),
		},
		{
			dst:  New(MyString("")),
			src:  MyRunes{0x266b, 0x266c},
			want: New(MyString("♫♬")),
		},

		// zero values
		{
			dst:  New(MyString("")),
			src:  []rune{},
			want: New(MyString("")),
		},
		{
			dst:  New(MyString("")),
			src:  []rune(nil),
			want: New(MyString("")),
		},
		{
			dst:  New(MyString("")),
			src:  []MyRune{},
			want: New(MyString("")),
		},
		{
			dst:  New(MyString("")),
			src:  []MyRune(nil),
			want: New(MyString("")),
		},
		{
			dst:  New(MyString("")),
			src:  MyRunes{},
			want: New(MyString("")),
		},
		{
			dst:  New(MyString("")),
			src:  MyRunes(nil),
			want: New(MyString("")),
		},
	}

	testDeepMap(t, tests...)
}

func TestMapMap(t *testing.T) {
	t.Parallel()

	type T struct{ A int }
	type T2 struct {
		A string
		B T
		c int
	}
	type T3 struct {
		A    T2
		B, C T
	}
	test := test{
		dst: &T3{A: T2{A: "foo"}},
		src: map[string]any{
			"a": map[string]any{
				"a": "bar",
				"b": map[string]any{"a": 42},
				"c": 1,
			},
			"b": &T{144}, // Mapping a reference
			"c": T{3},
			"d": T{299}, // Mapping a missing field (d doesn't exist)
		},
		mergeOpts: Options{WithOverwrite()},

		want: &T3{
			A: T2{
				A: "bar",
				B: T{42},
				c: 0,
			},
			B: T{144},
			C: T{3},
		},
		cmpOpts: cmp.Options{cmp.AllowUnexported(T2{})},
	}

	testDeepMap(t, test)
}

func TestSimpleMap(t *testing.T) {
	t.Parallel()

	type T struct{ A int }
	test := test{
		dst:  &T{},
		src:  map[string]any{"a": 42},
		want: &T{42},
	}

	testDeepMap(t, test)
}

func TestIfcMap(t *testing.T) {
	t.Parallel()

	type T struct{ I any }
	tests := []test{
		{
			dst:  &T{},
			src:  T{42},
			want: &T{42},
		},
		{
			name: "NonOverwrite",
			dst:  &T{23},
			src:  T{42},
			want: &T{23},
		},
		{
			name:      "WithOverwrite",
			dst:       &T{23},
			src:       T{42},
			mergeOpts: Options{WithOverwrite()},
			want:      &T{42},
		},
	}

	testDeepMap(t, tests...)
}

func TestBackAndForth(t *testing.T) {
	t.Parallel()

	type T struct{ A int }
	type T2 struct {
		A int
		T *T
		c int
	}
	dst := make(map[string]any)
	tests := []test{
		{
			dst: &dst,
			src: T2{42, &T{66}, 1},
			want: &map[string]any{
				"a": 42,
				"t": &T{66},
			},
		},
		{
			dst:     &T2{},
			src:     dst,
			want:    &T2{42, &T{66}, 0},
			cmpOpts: cmp.Options{cmp.AllowUnexported(T2{})},
		},
	}

	testDeepMap(t, tests...)
}

func TestEmbeddedPointerUnpacking(t *testing.T) {
	t.Parallel()

	type T struct{ A int }
	type T2 struct {
		A int
		T *T
		c int
	}

	newValue := 77
	var src = map[string]any{
		"t": map[string]any{"a": newValue},
	}
	tests := []test{
		{
			dst:       &T2{42, nil, 1},
			src:       src,
			mergeOpts: Options{WithOverwrite()},
			want:      &T2{42, &T{newValue}, 1},
			cmpOpts:   cmp.Options{cmp.AllowUnexported(T2{})},
		},
		{
			dst:       &T2{42, &T{66}, 1},
			src:       src,
			mergeOpts: Options{WithOverwrite()},
			want:      &T2{42, &T{newValue}, 1},
			cmpOpts:   cmp.Options{cmp.AllowUnexported(T2{})},
		},
	}

	testDeepMap(t, tests...)
}

func TestMapTime(t *testing.T) {
	t.Parallel()

	type T struct{ Created time.Time }
	now := time.Now()
	test := test{
		dst:  &T{},
		src:  map[string]any{"created": now},
		want: &T{now},
	}

	testDeepMap(t, test)
}

func TestNestedPtrValueInMap(t *testing.T) {
	t.Parallel()

	type T struct{ A int }
	type T2 struct{ M map[string]*T }
	test := test{
		dst:  &T2{map[string]*T{"x": {}}},
		src:  T2{map[string]*T{"x": {23}}},
		want: &T2{map[string]*T{"x": {23}}},
	}

	testDeepMap(t, test)
}

func TestIssue84MergeMapWithNilValueToStructWithOverride(t *testing.T) {
	t.Parallel()

	type T struct{ A, B, C int }
	test := test{
		dst:       &T{1, 2, 3},
		src:       map[string]any{"a": 4, "b": 5, "c": 6},
		mergeOpts: Options{WithOverwrite()},
		want:      &T{4, 5, 6},
	}

	testDeepMap(t, test)
}

func TestIssue84MergeMapWithoutKeyExistsToStructWithOverride(t *testing.T) {
	t.Parallel()

	type T struct{ A, B, C int }
	test := test{
		dst:       &T{1, 2, 3},
		src:       map[string]any{"a": 4, "b": 5},
		mergeOpts: Options{WithOverwrite()},
		want:      &T{4, 5, 3},
	}

	testDeepMap(t, test)
}

func TestIssue100(t *testing.T) {
	t.Parallel()

	type T struct{ I any }
	test := test{
		dst:  &T{},
		src:  map[string]any{"i": 23},
		want: &T{23},
	}

	testDeepMap(t, test)
}

func TestIssue138(t *testing.T) {
	t.Parallel()

	const js = `{"Port": 80}`

	var m = make(map[string]any)
	// encoding/json unmarshals numbers as float64
	// https://golang.org/pkg/encoding/json/#Unmarshal
	if err := json.Unmarshal([]byte(js), &m); err != nil {
		t.Fatal(err)
	}

	tests := []test{
		{
			dst:  &struct{ Port int }{},
			src:  m,
			want: &struct{ Port int }{80},
		},
		{
			dst:  &struct{ Port float64 }{},
			src:  m,
			want: &struct{ Port float64 }{80},
		},
	}

	testDeepMap(t, tests...)
}

func TestIssue143(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			dst: &map[string]any{
				"foo": map[string]any{
					"bar": []int{1, 2, 3},
				},
			},
			src: map[string]any{
				"foo": map[string]any{
					"bar": 23,
				},
			},
			mergeOpts: Options{WithOverwrite()},
			check: func(t testing.TB, dst any) {
				foo := (*dst.(*map[string]any))["foo"].(map[string]any)
				if _, ok := foo["bar"].(int); !ok {
					t.Error("expected int")
				}
			},
		},
		{
			dst: &map[string]any{
				"foo": map[string]any{
					"bar": []int{1, 2, 3},
				},
			},
			src: map[string]any{
				"foo": map[string]any{
					"bar": 23,
				},
			},
			check: func(t testing.TB, dst any) {
				foo := (*dst.(*map[string]any))["foo"].(map[string]any)
				if _, ok := foo["bar"].([]int); !ok {
					t.Errorf("expected []int")
				}
			},
		},
	}

	testDeepMap(t, tests...)
}

func TestMapMapInterfaceWithMultipleLayer(t *testing.T) {
	t.Parallel()

	test := test{
		dst: &map[string]any{
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

		want: &map[string]any{
			"k1": map[string]any{
				"k1.1": "v2",
				"k1.2": "v3",
			},
		},
	}

	testDeepMap(t, test)
}

func TestV039Issue152(t *testing.T) {
	t.Parallel()

	testDeepMap(t, test{
		dst: &map[string]any{
			"properties": map[string]any{
				"field1": map[string]any{
					"type": "text",
				},
				"field2": "ohai",
			},
		},
		src: map[string]any{
			"properties": map[string]any{
				"field1": "wrong",
			},
		},
		mergeOpts: Options{WithOverwrite()},
	})
}
