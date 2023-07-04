package merge_test

import (
	"reflect"
	"testing"
	"time"

	. "github.com/weiwenchen2022/merge"

	"github.com/google/go-cmp/cmp"
)

func New[T any](v T) *T { return &v }

type test struct {
	name string

	dst, src  any
	mergeOpts Options

	wantErr bool
	want    any
	cmpOpts cmp.Options

	check func(t testing.TB, dst any)
}

func makeDst(t testing.TB, dst any) any {
	if dst == nil {
		return nil
	}

	// t.Logf("%T", dst)

	vdst := reflect.ValueOf(dst)
	switch vdst.Kind() {
	case reflect.Slice:
		if vdst.IsNil() {
			break
		}

		v := reflect.MakeSlice(vdst.Type(), vdst.Len(), vdst.Cap())
		reflect.Copy(v, vdst)
		vdst = v
	case reflect.Map:
		if vdst.IsNil() {
			break
		}

		v := reflect.MakeMapWithSize(vdst.Type(), vdst.Len())
		for it := vdst.MapRange(); it.Next(); {
			v.SetMapIndex(it.Key(), it.Value())
		}
		vdst = v
	case reflect.Pointer:
		v := reflect.New(vdst.Type().Elem()).Elem()
		v.Set(vdst.Elem())
		vdst = v.Addr()
	case reflect.Struct:
		vdst = reflect.New(vdst.Type()).Elem()
	default:
		t.Fatal(vdst.Kind())
	}
	return vdst.Interface()
}

func testDeepMerge(t *testing.T, tests ...test) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			// ss := fmt.Sprintf("%#v", tt.src)

			dst := makeDst(t, tt.dst)
			if err := DeepMerge(dst, tt.src, tt.mergeOpts...); (err != nil) != tt.wantErr {
				if err == nil {
					t.Fatalf("want error got nil")
				} else {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Log(err)
			}

			// if got := fmt.Sprintf("%#v", tt.src); ss != got {
			// 	t.Error(cmp.Diff(ss, got))
			// }

			if tt.wantErr || tt.want == nil {
				return
			}

			if !cmp.Equal(tt.want, dst, tt.cmpOpts...) {
				t.Error(cmp.Diff(tt.want, dst, tt.cmpOpts))
			}

			if tt.check != nil {
				tt.check(t, dst)
			}
		})
	}
}

func TestBasicTypes(t *testing.T) {
	t.Parallel()

	one, two := (*int)(nil), new(int)
	*two = 2
	oneAgain, twoAgain := new(int), new(int)
	*oneAgain, *twoAgain = 1, 2

	tests := []test{
		{dst: &one, src: 1, want: &oneAgain},
		{dst: &two, src: 1, want: &twoAgain},
		{dst: New(""), src: "foo", want: New("foo")},
		{dst: New("foo"), src: "bar", want: New("foo")},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestSlices(t *testing.T) {
	t.Parallel()

	tests := []test{
		{dst: New([]string(nil)), src: []string{"foo"}, want: New([]string{"foo"})},
		{dst: make([]string, 1), src: []string{"foo"}, want: []string{"foo"}},

		{dst: []string{"foo"}, src: []string{"bar"}, want: []string{"foo"}},
		{dst: New([]string{"foo"}), src: []string{"foo", "bar"}, want: New([]string{"foo", "bar"})},

		{dst: []string{}, src: []string(nil), want: []string{}},
		{dst: ([]string(nil)), src: []string{}, want: []string(nil)},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestMaps(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			dst:  make(map[string][]int),
			src:  map[string][]int{"foo": {1, 2, 3}},
			want: map[string][]int{"foo": {1, 2, 3}},
		},
		{
			dst:  map[string][]int{},
			src:  map[string][]int{"foo": {1, 2, 3}},
			want: map[string][]int{"foo": {1, 2, 3}},
		},
		{
			dst:  New(map[string][]int(nil)),
			src:  map[string][]int{"foo": {1, 2, 3}},
			want: New(map[string][]int{"foo": {1, 2, 3}}),
		},
		{
			dst:  New(map[string][]int{}),
			src:  map[string][]int{"foo": {1, 2, 3}},
			want: New(map[string][]int{"foo": {1, 2, 3}}),
		},
		{
			dst:  map[string][]int{"foo": {1, 2, 3}},
			src:  map[string][]int{"foo": {1, 2, 3, 4}},
			want: map[string][]int{"foo": {1, 2, 3, 4}},
		},
		{
			dst:  map[string][]int(nil),
			src:  map[string][]int{},
			want: map[string][]int(nil),
		},
		{
			dst:  map[string][]int{},
			src:  map[string][]int(nil),
			want: map[string][]int{},
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestPointers(t *testing.T) {
	t.Parallel()

	one, oneAgain, two, twoAgain := (*int)(nil), new(int), new(int), new(int)
	*oneAgain, *two, *twoAgain = 1, 2, 2

	var pt *time.Time
	now := time.Now()
	pnow := &now

	tests := []test{
		{dst: &one, src: oneAgain, want: &oneAgain},
		{dst: &two, src: oneAgain, want: &twoAgain},
		{dst: &pt, src: now, want: &pnow},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestFunctions(t *testing.T) {
	t.Parallel()

	f := func() {}
	f1 := f
	opt := cmp.Comparer(func(f1, f2 func()) bool {
		return reflect.ValueOf(f1).UnsafePointer() == reflect.ValueOf(f2).UnsafePointer()
	})

	tests := []test{
		{dst: New((func())(nil)), src: f, want: &f, cmpOpts: cmp.Options{opt}},
		{dst: &f, src: func() {}, want: &f1, cmpOpts: cmp.Options{opt}},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestArrays(t *testing.T) {
	t.Parallel()

	tests := []test{
		{dst: New([...]int{2: 0}), src: [...]int{1, 2, 3}, want: New([...]int{1, 2, 3})},
		{dst: New([...]int{1, 2, 0}), src: [...]int{1, 2, 3}, want: New([...]int{1, 2, 3})},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestChannels(t *testing.T) {
	t.Parallel()

	ch1, ch2 := (chan bool)(nil), make(chan bool)
	ch3 := ch2

	tests := []test{
		{dst: &ch1, src: ch2, want: &ch2},
		{dst: &ch2, src: make(chan bool), want: &ch3},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}

func TestInterfaces(t *testing.T) {
	t.Parallel()

	one := 1
	var iface1 any = &one

	test := test{
		dst:  New(any(nil)),
		src:  &iface1,
		want: &iface1,
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, test) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, test) })
}

func TestMergeErrors(t *testing.T) {
	t.Parallel()

	type mystring string

	tests := []test{
		{dst: New(0), src: 1.0, wantErr: true, want: New(1)},                            // different types
		{dst: New(mystring("")), src: "foo", wantErr: true, want: New(mystring("foo"))}, // different types
		{wantErr: true}, // all zero values
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) {
		for i, tt := range tests {
			if tt.dst == nil && tt.src == nil {
				continue
			}
			tests[i].wantErr = false
		}
		testDeepMap(t, tests...)
	})
}

func TestCycles(t *testing.T) {
	t.Parallel()

	type CycleSlice []CycleSlice
	cycleSlice := make(CycleSlice, 1)
	cycleSlice[0] = cycleSlice

	type CyclePtr *CyclePtr
	var cyclePtr CyclePtr
	cyclePtr = &cyclePtr

	tests := []test{
		// slice cycles
		{
			dst:  New(CycleSlice(nil)),
			src:  cycleSlice,
			want: &cycleSlice,
			cmpOpts: cmp.Options{cmp.Comparer(func(c1, c2 CycleSlice) bool {
				return reflect.DeepEqual(c1, c2)
			})},
		},

		// pointer cycles
		{
			dst:  New(CyclePtr(nil)),
			src:  cyclePtr,
			want: &cyclePtr,
			cmpOpts: cmp.Options{cmp.Comparer(func(p1, p2 CyclePtr) bool {
				return reflect.DeepEqual(p1, p2)
			})},
		},
	}

	t.Run("Merge", func(t *testing.T) { testDeepMerge(t, tests...) })

	t.Run("Map", func(t *testing.T) { testDeepMap(t, tests...) })
}
