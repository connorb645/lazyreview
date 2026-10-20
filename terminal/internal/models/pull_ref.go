package models

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var (
	pullHost  = regexp.MustCompile(`^[A-Za-z0-9.-]+(:\d+)?$`)
	pullName  = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	pullLink  = regexp.MustCompile(`^https?://([^/]+)/([^/]+)/([^/]+)/pull/(\d+)(?:[/?#].*)?$`)
	pullID    = regexp.MustCompile(`^([^/]+\.[^/]+)/([^/]+)/([^/]+)/(\d+)$`)
	pullShort = regexp.MustCompile(`^([^/#\s]+)/([^/#\s]+)#(\d+)$`)
)

// PullRef names one pull request: its host, its repository as owner/name
// and its number.
type PullRef struct {
	Host string
	Repo string
	Pull int
}

// NewPullRef is pull request pull of repo, owner/name, on host, or an error
// saying these name no pull request.
func NewPullRef(tr *i18n.TranslationSet, host, repo string, pull int) (PullRef, error) {
	parts := strings.Split(repo, "/")
	if !pullHost.MatchString(host) || len(parts) != 2 || !pullName.MatchString(parts[0]) || !pullName.MatchString(parts[1]) ||
		parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." || pull <= 0 {
		return PullRef{}, refError(tr.NotAPull, map[string]string{"host": host, "repo": repo, "pull": strconv.Itoa(pull)})
	}
	return PullRef{Host: strings.ToLower(host), Repo: repo, Pull: pull}, nil
}

// ParsePullRef reads a pull request's address, such as
// https://github.com/octo/widgets/pull/12, anything after the number left
// out; a review id such as github.com/octo/widgets/12; or the short form
// octo/widgets#12, which means github.com.
func ParsePullRef(tr *i18n.TranslationSet, text string) (PullRef, error) {
	given := strings.TrimSpace(text)
	read := func(host, owner, name, number string) (PullRef, error) {
		pull, err := strconv.Atoi(number)
		if err != nil {
			pull = 0
		}
		return NewPullRef(tr, host, owner+"/"+name, pull)
	}
	if m := pullLink.FindStringSubmatch(given); m != nil {
		return read(m[1], m[2], m[3], m[4])
	}
	if m := pullID.FindStringSubmatch(given); m != nil {
		return read(m[1], m[2], m[3], m[4])
	}
	if m := pullShort.FindStringSubmatch(given); m != nil {
		return read("github.com", m[1], m[2], m[3])
	}
	return PullRef{}, refError(tr.NotAPullAddress, map[string]string{"text": given})
}

// ID is the review id the pull request gives: its review's folder under the
// data home, and its address in the app.
func (r PullRef) ID() string {
	return r.Host + "/" + r.Repo + "/" + strconv.Itoa(r.Pull)
}

// URL is the pull request's address on its host.
func (r PullRef) URL() string {
	return "https://" + r.Host + "/" + r.Repo + "/pull/" + strconv.Itoa(r.Pull)
}

// WithPull is the same repository's pull request pull.
func (r PullRef) WithPull(pull int) PullRef {
	return PullRef{Host: r.Host, Repo: r.Repo, Pull: pull}
}

func refError(words string, values map[string]string) error {
	return Refused(utils.ResolvePlaceholderString(words, values))
}
