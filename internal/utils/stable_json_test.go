package utils

import "testing"

func TestStableJSONSortsKeysAndWritesTextAsJavaScriptDoes(t *testing.T) {
	got, err := StableJSON(map[string]any{"b": []any{1, "x"}, "a": "say \"hi\"\n\t<&>   é \x01", "c": nil, "d": true})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a":"say \"hi\"\n\t<&> ` + " " + ` é \u0001","b":[1,"x"],"c":null,"d":true}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}
