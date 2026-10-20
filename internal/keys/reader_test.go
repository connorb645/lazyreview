package keys

import "testing"

func read(t *testing.T, b Bindings, keys ...string) (Reader, Result) {
	t.Helper()
	var r Reader
	var res Result
	for _, k := range keys {
		r, res = r.Read(k, b)
	}
	return r, res
}

func TestSequencesCountsAndTheLeader(t *testing.T) {
	b := Bindings{"j": "down", "g g": "top", "G": "bottom", "] c": "next hunk", "space l": "links", "space p": "properties", "d d": "drop"}
	cases := []struct {
		keys    []string
		action  string
		count   int
		waiting bool
	}{
		{[]string{"j"}, "down", 1, false},
		{[]string{"5", "j"}, "down", 5, false},
		{[]string{"1", "2", "j"}, "down", 12, false},
		{[]string{"g"}, "", 0, true},
		{[]string{"g", "g"}, "top", 1, false},
		{[]string{"3", "]", "c"}, "next hunk", 3, false},
		{[]string{"space", "l"}, "links", 1, false},
		{[]string{"g", "x"}, "", 0, false},
		{[]string{"0"}, "", 0, false},
	}
	for _, c := range cases {
		_, res := read(t, b, c.keys...)
		if res.Action != c.action || res.Count != c.count || res.Waiting != c.waiting {
			t.Errorf("%v: got %+v, want action %q count %d waiting %v", c.keys, res, c.action, c.count, c.waiting)
		}
	}
	if r, _ := read(t, b, "4", "g"); r.Typed() != "4g" || r.Pending() != "g" {
		t.Errorf("typed %q pending %q, want 4g and g", r.Typed(), r.Pending())
	}
	if r, _ := read(t, b, "g", "x"); r.Typed() != "" {
		t.Errorf("a sequence that means nothing is let go, but %q stayed", r.Typed())
	}
	choose := Bindings{"1": "first", "2": "second", "j": "down"}
	if _, res := read(t, choose, "1"); res.Action != "first" {
		t.Errorf("a digit the bindings take is no count: %+v", res)
	}
	if _, res := read(t, choose, "2", "1"); res.Action != "first" {
		t.Errorf("each bound digit is its own key: %+v", res)
	}
	after := b.After("space")
	if len(after) != 2 || after[0] != [2]string{"l", "links"} || after[1] != [2]string{"p", "properties"} {
		t.Errorf("after the leader: %v", after)
	}
}
