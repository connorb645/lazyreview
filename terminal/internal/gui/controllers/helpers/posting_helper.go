package helpers

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var postOrder = []string{"comment", "approve-with-comments", "approve"}

// PostingHelper posts a pull request's queued comments to GitHub, as a
// comment or an approval.
type PostingHelper struct {
	c *HelperCommon
}

// QueueMember is the queue's part for a pull request of the review.
func (self *PostingHelper) QueueMember(s context.ContextTree, pull int) (models.QueueMember, bool) {
	if s.Review == nil || s.Review.State.Queue == nil {
		return models.QueueMember{}, false
	}
	for _, m := range s.Review.State.Queue.Members {
		if m.Pull == pull {
			return m, true
		}
	}
	return models.QueueMember{}, false
}

// Open is the state with posting's choice open for a pull request, or the
// only one queued when none is named, or why nothing can be posted.
func (self *PostingHelper) Open(s context.ContextTree, pull int) context.ContextTree {
	tr := self.c.Tr
	if s.Review == nil || s.Review.State.Queue == nil {
		return s.Said(tr.NoQueueToPost, true)
	}
	if s.Review.State.Queue.Moved != nil {
		return s.Said(tr.ReviewMovedNoSend, true)
	}
	if members := s.Review.State.Queue.Members; pull == 0 && len(members) == 1 {
		pull = members[0].Pull
	}
	m, ok := self.QueueMember(s, pull)
	if !ok {
		return s.Said(utils.ResolvePlaceholderString(tr.PostsAtCursor, map[string]string{"post": self.c.KeyLabels(s.Place()).Of("post")}), true)
	}
	pn := context.PostingContext{Pull: m.Pull}
	for _, action := range postOrder {
		if _, offered := m.Sets[action]; offered {
			pn.Actions = append(pn.Actions, action)
		}
	}
	if len(pn.Actions) == 0 {
		return s.Said(utils.ResolvePlaceholderString(tr.PullWasState, map[string]string{"pull": utils.Itoa(m.Pull), "state": m.State}), true)
	}
	s = s.Opened(context.PostingKey)
	s.Posting = pn
	return s
}

// Post is the state sending the chosen action to GitHub, and the command
// that sends it.
func (self *PostingHelper) Post(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	pn := s.Posting
	s = s.WithoutPosting()
	m, _ := self.QueueMember(s, pn.Pull)
	action := pn.Actions[pn.Choice]
	client, id, build, set := self.c.Reviews, s.Review.ID, s.Review.Page.Build, m.Sets[action].Set
	return s.Said(self.c.Tr.CheckingThenSending, false), func() tea.Msg {
		message, state, err := client.Post(id, pn.Pull, action, set, build)
		return types.Posted{Pull: pn.Pull, Message: message, State: state, Err: err}
	}
}

// AfterPost is the state once GitHub took what was posted, or why it did
// not, shown in a popup.
func (self *PostingHelper) AfterPost(s context.ContextTree, msg types.Posted) context.ContextTree {
	tr := self.c.Tr
	title, words, notice := tr.SentToGitHub, msg.Message, tr.SentHowToClose
	if msg.Err != nil {
		title = utils.ResolvePlaceholderString(tr.PostTo, map[string]string{"repo": s.Review.State.Repo, "pull": utils.Itoa(msg.Pull)})
		words, notice = msg.Err.Error(), tr.NothingPostedHowToClose
	} else {
		s = self.c.Helpers.Refresh.WithState(s, msg.State)
	}
	pn := context.MenuContext{Menu: presentation.Menu{Title: title, Rows: []presentation.MenuRow{{Right: strings.ToUpper(words[:1]) + words[1:]}}}}
	closing := map[string]string{"close": self.c.KeyLabels(context.MenuKey).Of("close")}
	return s.WithMenu(pn).Said(utils.ResolvePlaceholderString(notice, closing), msg.Err != nil)
}
