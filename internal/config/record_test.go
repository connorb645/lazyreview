package config

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

const (
	recordedPath = "/home/lee/.config/lazyreview/config.yml"
	recordedHome = "/home/lee"
)

type recordedConfigs struct {
	Configs []struct {
		Text    string            `json:"text"`
		Error   string            `json:"error"`
		Values  json.RawMessage   `json:"values"`
		Sources map[string]string `json:"sources"`
		Unused  []string          `json:"unused"`
		Part    json.RawMessage   `json:"part"`
	} `json:"configs"`
	Edits []struct {
		Text   string     `json:"text"`
		Change ListChange `json:"change"`
		Out    *string    `json:"out"`
		Error  string     `json:"error"`
	} `json:"edits"`
	Scalars []struct {
		Text   string `json:"text"`
		Type   string `json:"type"`
		Shown  string `json:"shown"`
		String string `json:"string"`
	} `json:"scalars"`
}

func recordedReadings(t *testing.T) recordedConfigs {
	t.Helper()
	var r recordedConfigs
	text, err := os.ReadFile(testsupport.Testdata("configs.json"))
	if err == nil {
		err = json.Unmarshal(text, &r)
	}
	if err != nil {
		t.Fatalf("could not read the recorded configs: %v", err)
	}
	return r
}

func plain(t *testing.T, v any) any {
	t.Helper()
	text, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(text, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// Each plain scalar reads in Go as lazyreview's Node reader read it, by
// the core schema.
func TestAPlainScalarReadsAsTheNodeReaderReadIt(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, s := range recordedReadings(t).Scalars {
		given, err := ParseYAML(tr, "a: "+s.Text)
		if err != nil {
			t.Errorf("%q: %v", s.Text, err)
			continue
		}
		v := given.(YAMLMap).Values["a"]
		kind := map[bool]string{true: "null"}[v == nil]
		switch v.(type) {
		case bool:
			kind = "boolean"
		case float64:
			kind = "number"
		case string:
			kind = "string"
		}
		if kind != s.Type || Shown(v) != s.Shown || jsText(v) != s.String {
			t.Errorf("%q reads as %s %s %q, not %s %s %q", s.Text, kind, Shown(v), jsText(v), s.Type, s.Shown, s.String)
		}
	}
}

// Each recorded config reads in Go to what Node read, values, sources,
// unused settings and the terminal app's part alike, or is refused in the
// same words. Only a refusal for YAML the parser cannot read may say why in
// other words.
func TestEachConfigReadsAsTheNodeReaderReadIt(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	none := func(string) string { return "" }
	for _, c := range recordedReadings(t).Configs {
		read, err := ConfigFrom(tr, c.Text, recordedPath, recordedHome)
		if c.Error != "" {
			syntax := strings.HasPrefix(c.Error, recordedPath+" could not be read: ") && !strings.Contains(c.Error, tr.ConfigDuplicatedKey)
			switch {
			case err == nil:
				t.Errorf("%q was read, where Node refused it: %s", c.Text, c.Error)
			case syntax && !strings.HasPrefix(err.Error(), recordedPath+" could not be read: "):
				t.Errorf("%q was refused as %q, not as unreadable", c.Text, err)
			case !syntax && err.Error() != c.Error:
				t.Errorf("%q was refused as:\n%s\nnot as:\n%s", c.Text, err, c.Error)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q was refused: %v", c.Text, err)
			continue
		}
		var want any
		_ = json.Unmarshal(c.Values, &want)
		if got := plain(t, read.Values); !reflect.DeepEqual(got, want) {
			t.Errorf("%q reads as\n%v\nnot\n%v", c.Text, got, want)
		}
		if !reflect.DeepEqual(read.Sources, c.Sources) || !reflect.DeepEqual(read.Unused, c.Unused) {
			t.Errorf("%q gives sources %v and unused %v", c.Text, read.Sources, read.Unused)
		}
		part, err := TerminalPart(tr, read, none)
		_ = json.Unmarshal(c.Part, &want)
		if got := plain(t, part); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%q hands the terminal app\n%v\nnot\n%v (%v)", c.Text, got, want, err)
		}
	}
}

func equalJSON(t *testing.T, a, b any) bool {
	t.Helper()
	return reflect.DeepEqual(plain(t, a), plain(t, b))
}

// Each recorded change to the hidden repositories edits the text in Go as
// Node edited it, to the byte, or is refused in the same words.
func TestEachListEditIsMadeAsNodeMadeIt(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, e := range recordedReadings(t).Edits {
		out, err := ListEdited(tr, e.Text, "github.hiddenRepositories", e.Change)
		switch {
		case e.Error != "" && (err == nil || err.Error() != e.Error):
			t.Errorf("%q %+v gave %q, %v, not the refusal %q", e.Text, e.Change, out, err, e.Error)
		case e.Error == "" && (err != nil || out != *e.Out):
			t.Errorf("%q %+v gave\n%q, %v\nnot\n%q", e.Text, e.Change, out, err, *e.Out)
		}
	}
}
