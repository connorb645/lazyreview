package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func TestStopEndsTheServerAnOlderLazyreviewLeftRunningByTheProcessIDItRecorded(t *testing.T) {
	record := filepath.Join(t.TempDir(), "server.json")
	old := NewOldServer(i18n.EnglishTranslationSet(), record)
	if running, err := old.Stop(); running || err != nil {
		t.Fatalf("with no record, stop says %v, %v", running, err)
	}
	server := exec.Command("sleep", "60")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	ended := make(chan struct{})
	go func() {
		_ = server.Wait()
		close(ended)
	}()
	defer server.Process.Kill()
	text := `{"pid":` + strconv.Itoa(server.Process.Pid) + `,"address":"http://127.0.0.1:4747","token":"x"}`
	if err := os.WriteFile(record, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	if running, err := old.Stop(); !running || err != nil {
		t.Fatalf("stop says %v, %v; want the old server stopped", running, err)
	}
	<-ended
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatal("the old server's record is still there")
	}
	if err := os.WriteFile(record, []byte(`{"pid":`+strconv.Itoa(server.Process.Pid)+`}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if running, err := old.Stop(); running || err != nil {
		t.Fatalf("a record of a server that has gone says %v, %v", running, err)
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatal("the record of a server that has gone is still there")
	}
}

func TestARecordThatIsNotJSONStopsNothingAndIsNamed(t *testing.T) {
	record := filepath.Join(t.TempDir(), "server.json")
	if err := os.WriteFile(record, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewOldServer(i18n.EnglishTranslationSet(), record).Stop()
	if err == nil || err.Error() != "could not read "+record+": unexpected end of JSON input" {
		t.Fatalf("an empty record says %v", err)
	}
}
