package app

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/connorb645/lazyreview/terminal/internal/commands/oscommands"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// OldServer is a server an older lazyreview may have left running, which
// recorded itself in the data home's server.json as {pid, address, token}.
type OldServer struct {
	tr        *i18n.TranslationSet
	record    string
	processes oscommands.ServerProcess
}

// NewOldServer is the server whose record is the file record.
func NewOldServer(tr *i18n.TranslationSet, record string) *OldServer {
	return &OldServer{tr: tr, record: record}
}

// Stop stops that server, by the process id it recorded only, removes its
// record, and says whether one was running.
func (s *OldServer) Stop() (bool, error) {
	text, err := os.ReadFile(s.record)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	var recorded map[string]any
	if err == nil {
		err = json.Unmarshal(text, &recorded)
	}
	if err != nil {
		return false, models.Refused(utils.ResolvePlaceholderString(s.tr.Unreadable, map[string]string{"path": s.record, "why": err.Error()}))
	}
	pid, _ := recorded["pid"].(float64)
	running := pid > 0 && s.processes.Alive(int(pid))
	if running {
		s.processes.Stop(int(pid))
	}
	if err := os.Remove(s.record); err != nil && !errors.Is(err, os.ErrNotExist) {
		return running, err
	}
	return running, nil
}
