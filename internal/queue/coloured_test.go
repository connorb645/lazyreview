package queue

import (
	"reflect"
	"testing"
)

func TestEachMemberKeepsItsColourAndOneNewToTheStackTakesTheFirstFree(t *testing.T) {
	for _, c := range []struct{ given, want []int }{
		{[]int{0, 0, 0}, []int{1, 2, 3}},
		{[]int{1, 0, 3}, []int{1, 2, 3}},
		{[]int{2, 0, 0}, []int{2, 1, 3}},
		{[]int{1, 3, 0}, []int{1, 3, 2}},
		{[]int{1, 2, 3, 4, 5, 6, 7, 8, 0, 0}, []int{1, 2, 3, 4, 5, 6, 7, 8, 1, 2}},
	} {
		if got := Coloured(c.given); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%v is coloured %v, want %v", c.given, got, c.want)
		}
	}
}
