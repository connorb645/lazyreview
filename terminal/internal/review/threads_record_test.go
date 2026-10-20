package review

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

func TestThreadsAreKeptAndPlacedAsNodeKeepsThem(t *testing.T) {
	var record struct {
		Threads []struct {
			Pull   int                         `json:"pull"`
			Stack  *models.StackRecord         `json:"stack"`
			Nodes  []models.GitHubReviewThread `json:"nodes"`
			GitHub string                      `json:"github"`
		} `json:"threads"`
	}
	text, err := os.ReadFile(testsupport.Testdata("load-parts.json"))
	if err == nil {
		err = json.Unmarshal(text, &record)
	}
	if err != nil {
		t.Fatalf("could not read the record: %v", err)
	}
	if len(record.Threads) == 0 {
		t.Fatal("the record holds no threads")
	}
	for _, c := range record.Threads {
		var out bytes.Buffer
		encoder := json.NewEncoder(&out)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(PlaceThreads(ReadThreads(c.Pull, c.Nodes), c.Stack)); err != nil {
			t.Fatal(err)
		}
		if out.String() != c.GitHub {
			t.Errorf("#%d's threads are kept as\n%s\nnot as Node keeps them:\n%s", c.Pull, out.String(), c.GitHub)
		}
	}
}
