package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func streaming(t *testing.T, streams ...[]string) *Client {
	t.Helper()
	connected := 0
	stand := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/reviews/github.com/octo/widgets/21/events" || r.Header.Get("Accept") != "text/event-stream" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "retry: 1000\n\n")
		if connected < len(streams) {
			for _, data := range streams[connected] {
				fmt.Fprintf(w, "data: %s\n\n", data)
			}
		}
		connected++
	}))
	t.Cleanup(stand.Close)
	file := filepath.Join(t.TempDir(), "server.json")
	record, _ := json.Marshal(Record{Address: stand.URL, Token: "test-token"})
	if err := os.WriteFile(file, record, 0o600); err != nil {
		t.Fatal(err)
	}
	return NewClient(file)
}

func TestEventsSendsEachNoticeAndSaysWhyItStopped(t *testing.T) {
	c := streaming(t, []string{`{"state":"aa","build":"bb"}`, `{"state":"cc"}`})
	notices := make(chan Notice, 4)
	err := c.Events(context.Background(), "github.com/octo/widgets/21", notices)
	close(notices)
	var got []Notice
	for n := range notices {
		got = append(got, n)
	}
	if len(got) != 2 || got[0] != (Notice{State: "aa", Build: "bb"}) || got[1] != (Notice{State: "cc"}) {
		t.Fatalf("the notices: %+v", got)
	}
	if err == nil || err.Error() != "the server ended the stream of changes" {
		t.Fatalf("it stopped with %v", err)
	}
}

func TestEventsStopsWhenAskedTo(t *testing.T) {
	c := streaming(t, []string{`{"state":"aa","build":"bb"}`})
	ctx, stop := context.WithCancel(context.Background())
	stop()
	if err := c.Events(ctx, "github.com/octo/widgets/21", make(chan Notice)); err == nil {
		t.Fatal("a stopped stream says nothing of why")
	}
}
