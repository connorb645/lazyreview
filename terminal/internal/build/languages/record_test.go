package languages

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

type recordedLanguages struct {
	Readings []struct {
		Language     string          `json:"language"`
		Text         string          `json:"text"`
		Masked       string          `json:"masked"`
		Declarations json.RawMessage `json:"declarations"`
		Imports      json.RawMessage `json:"imports"`
	} `json:"readings"`
	Masks []struct {
		Syntax string `json:"syntax"`
		Text   string `json:"text"`
		Masked string `json:"masked"`
	} `json:"masks"`
	BareNames []struct {
		Entry string `json:"entry"`
		Name  string `json:"name"`
	} `json:"bareNames"`
	Splits []struct {
		Text  string   `json:"text"`
		Parts []string `json:"parts"`
	} `json:"splits"`
	Identifiers []struct {
		Text  string   `json:"text"`
		Words []string `json:"words"`
	} `json:"identifiers"`
	Languages []struct {
		Path   string  `json:"path"`
		Name   *string `json:"name"`
		Family *string `json:"family"`
	} `json:"languages"`
}

func recordedReadings(t *testing.T) recordedLanguages {
	t.Helper()
	var r recordedLanguages
	text, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "languages.json"))
	if err == nil {
		err = json.Unmarshal(text, &r)
	}
	if err != nil {
		t.Fatalf("could not read the recorded readings: %v", err)
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

func recordedValue(t *testing.T, text json.RawMessage) any {
	t.Helper()
	var out any
	if err := json.Unmarshal(text, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// Each reader finds in each text the declarations, properties and imports
// lazyreview's Node reader found, at the same places, and masks the text
// as it did, one blank per byte.
func TestEachReaderReadsATextAsTheNodeReaderReadIt(t *testing.T) {
	for i, r := range recordedReadings(t).Readings {
		read := languages[slices.IndexFunc(languages, func(l Language) bool { return l.Name == r.Language })].Read(r.Text)
		if read.Masked != r.Masked {
			t.Errorf("%s reading %d masks the text as\n%q\nnot\n%q", r.Language, i, read.Masked, r.Masked)
		}
		if got, want := plain(t, read.Declarations), recordedValue(t, r.Declarations); !reflect.DeepEqual(got, want) {
			t.Errorf("%s reading %d of\n%s\nfinds declarations\n%v\nnot\n%v", r.Language, i, r.Text, got, want)
		}
		imports := read.Imports
		if imports == nil {
			imports = []Import{}
		}
		if got, want := plain(t, imports), recordedValue(t, r.Imports); !reflect.DeepEqual(got, want) {
			t.Errorf("%s reading %d finds imports %v, not %v", r.Language, i, got, want)
		}
	}
}

// Each language's syntax masks comments and the insides of strings as the
// Node masking did.
func TestEachSyntaxMasksATextAsTheNodeMaskingDid(t *testing.T) {
	syntaxes := map[string]syntax{"swift": swiftSyntax, "kotlin": kotlinSyntax, "java": javaSyntax, "typescript": typescriptSyntax, "python": pythonSyntax}
	for _, m := range recordedReadings(t).Masks {
		if got := maskSource(m.Text, syntaxes[m.Syntax]); got != m.Masked {
			t.Errorf("%s masks %q as %q, not %q", m.Syntax, m.Text, got, m.Masked)
		}
	}
}

// A header's entries, its bare type names and the identifiers of a piece
// of code read as Node read them, and a path chooses the same language.
func TestTheHeaderHelpersAndTheLanguagesAgreeWithNode(t *testing.T) {
	r := recordedReadings(t)
	for _, b := range r.BareNames {
		if got := bareTypeName(b.Entry); got != b.Name {
			t.Errorf("the bare type name of %q is %q, not %q", b.Entry, got, b.Name)
		}
	}
	for _, s := range r.Splits {
		if got := splitTopLevel(s.Text); !reflect.DeepEqual(got, s.Parts) {
			t.Errorf("%q splits into %q, not %q", s.Text, got, s.Parts)
		}
	}
	for _, w := range r.Identifiers {
		if got := Identifiers(w.Text); !reflect.DeepEqual(got, w.Words) {
			t.Errorf("the identifiers of %q are %q, not %q", w.Text, got, w.Words)
		}
	}
	for _, l := range r.Languages {
		language := LanguageFor(l.Path)
		switch {
		case language == nil && l.Name != nil:
			t.Errorf("no language reads %q, but %s does", l.Path, *l.Name)
		case language != nil && (l.Name == nil || language.Name != *l.Name || language.Family != *l.Family):
			t.Errorf("%s reads %q, not the language Node chose", language.Name, l.Path)
		}
	}
}
