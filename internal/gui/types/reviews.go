package types

import (
	"context"

	"github.com/connorb645/lazyreview/internal/models"
)

// Reviews is what the app reads and changes reviews through: a review's
// state, its page and each file; where a comment goes, the queue and
// posting; Claude's conversations; loading; the open pull requests and the
// hidden repositories; and each review's change notices.
type Reviews interface {
	State(id string) (models.State, error)
	Page(id string) (models.Page, error)
	File(id, path, build string) (models.File, error)
	Where(id string, at models.Lines, pull int, build string) (models.Where, error)
	Queue(id string, at models.Lines, pull int, body, build string) (models.State, error)
	Edit(id string, comment int, body, version, build string) (models.State, error)
	Drop(id string, comment int, version, build string) (models.State, error)
	Post(id string, pull int, action, set, build string) (string, models.State, error)
	Ask(id string, at models.Lines, text, build string) (models.Conversation, error)
	Reply(id string, conversation int, text, build string) (models.Conversation, error)
	Stop(id string, conversation int) (models.Conversation, error)
	Refresh(id string) ([]string, error)
	Pulls() (models.Pulls, error)
	ChangeHidden(hide string, show, repos []string) (models.Hidden, error)
	Load(links []string, account string) (string, []string, error)
	Events(ctx context.Context, id string, notices chan<- models.Notice) error
}
