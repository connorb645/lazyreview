package cli

import (
	"regexp"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var (
	reviewAddress = regexp.MustCompile(`^(?:lazyreview://|https?://(?:127\.0\.0\.1|localhost)(?::\d+)?/review/)([^?#]+?)/*(?:[?#].*)?$`)
	webAddress    = regexp.MustCompile(`^https?://`)
)

// TerminalLink is what the text `terminal` is given names: a review, or a
// pull request that no review may hold yet.
type TerminalLink struct {
	Review models.ReviewRef
	Pull   *models.PullRef
}

// ReadTerminalLink reads the text `terminal` is given: a review's address,
// lazyreview://<review id> as load and list print it, or the address an
// older lazyreview printed on any port; a pull request's address; or a
// review id. It fails in one line when the text is none of these.
func ReadTerminalLink(tr *i18n.TranslationSet, text string) (TerminalLink, error) {
	notALink := models.Refused(utils.ResolvePlaceholderString(tr.CliNotALink, map[string]string{"text": text}))
	if m := reviewAddress.FindStringSubmatch(text); m != nil {
		ref, err := models.ParseReviewRef(tr, m[1])
		if err != nil {
			return TerminalLink{}, notALink
		}
		return TerminalLink{Review: ref}, nil
	}
	if webAddress.MatchString(text) {
		pull, err := models.ParsePullRef(tr, text)
		if err != nil {
			return TerminalLink{}, notALink
		}
		return TerminalLink{Pull: &pull}, nil
	}
	ref, err := models.ParseReviewRef(tr, text)
	if err != nil {
		return TerminalLink{}, notALink
	}
	return TerminalLink{Review: ref}, nil
}
