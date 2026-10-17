package utils

import "testing"

func TestPlaceholdersAreFilledByNameAndTheRestIsLeftAsItIs(t *testing.T) {
	cases := []struct {
		text   string
		values map[string]string
		want   string
	}{
		{"could not open {{.name}}", map[string]string{"name": "notes.md"}, "could not open notes.md"},
		{"{{.at}} of {{.total}}", map[string]string{"at": "2", "total": "9"}, "2 of 9"},
		{"{{name}} and {{.name}}", map[string]string{"name": "lee"}, "lee and lee"},
		{"{{.pulls}} only", map[string]string{"pull": "#21"}, "{{.pulls}} only"},
		{"no values", nil, "no values"},
	}
	for _, c := range cases {
		if got := ResolvePlaceholderString(c.text, c.values); got != c.want {
			t.Errorf("%q with %v is %q, not %q", c.text, c.values, got, c.want)
		}
	}
}
