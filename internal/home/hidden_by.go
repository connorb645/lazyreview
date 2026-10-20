// Package home is the terminal app's home screen: the open pull requests
// that involve the config's accounts, and the repositories it leaves out.
package home

import (
	"strings"
)

// HiddenBy is the entries of the config's hidden repositories, as the
// config writes them, that hide the repository name, written owner/name:
// the entry naming it, and owner/* for every repository of its owner. Case
// does not count, as it does not in GitHub's names.
func HiddenBy(entries []string, name string) []string {
	repo := strings.ToLower(name)
	owner, _, _ := strings.Cut(repo, "/")
	out := []string{}
	for _, entry := range entries {
		if e := strings.ToLower(entry); e == repo || e == owner+"/*" {
			out = append(out, entry)
		}
	}
	return out
}
