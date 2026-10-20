package queue

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

type recordedAbsorbed struct {
	Ref    string          `json:"ref"`
	Review json.RawMessage `json:"review"`
}

func TestRegroupingAnswersAsNodeDoes(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	var record struct {
		Regroups []struct {
			Given struct {
				Previous *models.Review                 `json:"previous"`
				Fresh    []models.ReviewMember          `json:"fresh"`
				Leaving  map[string]models.ReviewMember `json:"leaving"`
				Absorbed []recordedAbsorbed             `json:"absorbed"`
				Anchor   string                         `json:"anchor"`
				Now      string                         `json:"now"`
			} `json:"given"`
			Out *struct {
				Members json.RawMessage    `json:"members"`
				Earlier json.RawMessage    `json:"earlier"`
				Moved   []recordedAbsorbed `json:"moved"`
				Notes   []string           `json:"notes"`
			} `json:"out"`
			Error string `json:"error"`
		} `json:"regroups"`
	}
	text, err := os.ReadFile(testsupport.Testdata("load-parts.json"))
	if err == nil {
		err = json.Unmarshal(text, &record)
	}
	if err != nil {
		t.Fatalf("could not read the record: %v", err)
	}
	if len(record.Regroups) == 0 {
		t.Fatal("the record holds no regrouping")
	}
	for i, c := range record.Regroups {
		given := Regrouping{Previous: c.Given.Previous, Fresh: c.Given.Fresh, Leaving: map[int]models.ReviewMember{}, Anchor: c.Given.Anchor, Now: c.Given.Now}
		for pull, m := range c.Given.Leaving {
			n, _ := strconv.Atoi(pull)
			given.Leaving[n] = m
		}
		for _, a := range c.Given.Absorbed {
			ref, err := models.ParseReviewRef(tr, a.Ref)
			if err != nil {
				t.Fatal(err)
			}
			var review models.Review
			if err := json.Unmarshal(a.Review, &review); err != nil {
				t.Fatal(err)
			}
			given.Absorbed = append(given.Absorbed, Absorbed{Ref: ref, Review: review})
		}
		out, err := Regroup(tr, given)
		if c.Out == nil {
			if err == nil || err.Error() != c.Error {
				t.Errorf("regrouping %d: got %v, not Node's refusal %q", i, err, c.Error)
			}
			continue
		}
		if err != nil {
			t.Errorf("regrouping %d is refused: %v", i, err)
			continue
		}
		sameJSON(t, i, "members", out.Members, c.Out.Members)
		sameJSON(t, i, "earlier", out.Earlier, c.Out.Earlier)
		if !reflect.DeepEqual(out.Notes, c.Out.Notes) {
			t.Errorf("regrouping %d notes %q, not Node's %q", i, out.Notes, c.Out.Notes)
		}
		if len(out.Moved) != len(c.Out.Moved) {
			t.Errorf("regrouping %d moves %d reviews, not Node's %d", i, len(out.Moved), len(c.Out.Moved))
			continue
		}
		for j, m := range out.Moved {
			if m.Ref.ID() != c.Out.Moved[j].Ref {
				t.Errorf("regrouping %d moves %s, not %s", i, m.Ref.ID(), c.Out.Moved[j].Ref)
			}
			sameJSON(t, i, "moved review", m.Review, c.Out.Moved[j].Review)
		}
	}
}

func sameJSON(t *testing.T, i int, what string, got any, want json.RawMessage) {
	t.Helper()
	text, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	if err := json.Unmarshal(text, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("regrouping %d gives %s\n%s\nnot Node's\n%s", i, what, text, want)
	}
}
