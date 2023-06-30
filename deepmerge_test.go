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
	mergeopts Options

	wantErr bool
	want    any
	cmpopts cmp.Options
}

func testDeepMerge(t *testing.T, tests ...test) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			// ss := fmt.Sprintf("%#v", tt.src)
			if err := DeepMerge(tt.dst, tt.src, tt.mergeopts...); (err != nil) != tt.wantErr {
				t.Fatal(err)
			} else if err != nil {
				t.Log(err)
			}
			// if got := fmt.Sprintf("%#v", tt.src); ss != got {
			// 	t.Error(cmp.Diff(ss, got))
			// }
			if tt.want == nil {
				return
			}
			if !cmp.Equal(tt.want, tt.dst, tt.cmpopts...) {
				t.Error(cmp.Diff(tt.want, tt.dst, tt.cmpopts))
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
	testDeepMerge(t, tests...)
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
	testDeepMerge(t, tests...)
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
	testDeepMerge(t, tests...)
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
	testDeepMerge(t, tests...)
}

func TestFunctions(t *testing.T) {
	t.Parallel()

	f := func() {}
	f1 := f
	opt := cmp.Comparer(func(f1, f2 func()) bool {
		return reflect.ValueOf(f1).UnsafePointer() == reflect.ValueOf(f2).UnsafePointer()
	})

	tests := []test{
		{dst: New((func())(nil)), src: f, want: &f, cmpopts: cmp.Options{opt}},
		{dst: &f, src: func() {}, want: &f1, cmpopts: cmp.Options{opt}},
	}
	testDeepMerge(t, tests...)
}

func TestArrays(t *testing.T) {
	t.Parallel()

	tests := []test{
		{dst: New([...]int{2: 0}), src: [...]int{1, 2, 3}, want: New([...]int{1, 2, 3})},
		{dst: New([...]int{1, 2, 0}), src: [...]int{1, 2, 3}, want: New([...]int{1, 2, 3})},
	}
	testDeepMerge(t, tests...)
}

func TestChannels(t *testing.T) {
	t.Parallel()

	ch1, ch2 := (chan bool)(nil), make(chan bool)
	ch3 := ch2

	tests := []test{
		{dst: &ch1, src: ch2, want: &ch2},
		{dst: &ch2, src: make(chan bool), want: &ch3},
	}
	testDeepMerge(t, tests...)
}

func TestInterfaces(t *testing.T) {
	t.Parallel()

	one := 1
	var iface1 any = &one
	testDeepMerge(t, test{
		dst:  New(any(nil)),
		src:  &iface1,
		want: &iface1,
	})
}

func TestMergeErrors(t *testing.T) {
	t.Parallel()

	type mystring string

	tests := []test{
		{dst: New(1), src: 1.0, wantErr: true, want: New(1)},                               // different types
		{dst: New(mystring("foo")), src: "bar", wantErr: true, want: New(mystring("foo"))}, // different types
		{wantErr: true}, // all zero values
	}
	testDeepMerge(t, tests...)
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
			cmpopts: cmp.Options{cmp.Comparer(func(c1, c2 CycleSlice) bool {
				return reflect.DeepEqual(c1, c2)
			})},
		},

		// pointer cycles
		{
			dst:  New(CyclePtr(nil)),
			src:  cyclePtr,
			want: &cyclePtr,
			cmpopts: cmp.Options{cmp.Comparer(func(p1, p2 CyclePtr) bool {
				return reflect.DeepEqual(p1, p2)
			})},
		},
	}
	testDeepMerge(t, tests...)
}
