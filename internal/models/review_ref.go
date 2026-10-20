package models

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/connorb645/lazyreview/internal/i18n"
)

var digits = regexp.MustCompile(`^\d+$`)

// ReviewRef names a review: a pull request's, as a PullRef, or a branch's,
// as a BranchRef.
type ReviewRef interface {
	ID() string
}

// ParseReviewRef reads a review id, local/<repository key>/<branch> or
// <host>/<owner>/<repo>/<number>, each part percent-encoded or not, as
// text: a part that decodes to no text names no review.
func ParseReviewRef(tr *i18n.TranslationSet, id string) (ReviewRef, error) {
	parts := strings.Split(id, "/")
	for i, p := range parts {
		decoded, err := url.PathUnescape(p)
		if err != nil || !utf8.ValidString(decoded) {
			return nil, refError(tr.NotAReviewID, map[string]string{"id": id})
		}
		parts[i] = decoded
	}
	if parts[0] == "local" && len(parts) == 3 {
		return ParseBranchRef(tr, strings.Join(parts, "/"))
	}
	if len(parts) != 4 || !digits.MatchString(parts[3]) {
		return nil, refError(tr.NotAReviewID, map[string]string{"id": id})
	}
	pull, err := strconv.Atoi(parts[3])
	if err != nil {
		pull = 0
	}
	return NewPullRef(tr, parts[0], parts[1]+"/"+parts[2], pull)
}
