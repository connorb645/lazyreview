// Package github talks to GitHub through the gh command, as one account on
// one host.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	timeout     = 60 * time.Second
	pageSize    = 100
	searchPages = 10
)

var unreachable = regexp.MustCompile(`^error connecting to `)

// GitHub answers as one account on one host. With an account named, each
// call carries that account's own token, so the account gh has active for
// everything else on the machine never changes; with none, it answers as
// gh's active account. Every call has a time limit, and a failure says what
// GitHub refused, or that the host could not be reached.
type GitHub struct {
	tr      *i18n.TranslationSet
	host    string
	account string
	run     Run
	token   string
}

// New answers as account on host, running gh with run, or with gh itself
// when run is nil.
func New(tr *i18n.TranslationSet, host, account string, run Run) *GitHub {
	if run == nil {
		run = RunGh
	}
	return &GitHub{tr: tr, host: host, account: account, run: run}
}

// Environment is what a child process needs to act as this account, such as
// git fetching through gh.
func (g *GitHub) Environment() ([]string, error) {
	if g.account == "" {
		return nil, nil
	}
	if g.token == "" {
		done := g.run([]string{"auth", "token", "--hostname", g.host, "--user", g.account}, nil, nil)
		if done.Code != 0 {
			return nil, models.Refused(utils.ResolvePlaceholderString(g.tr.GitHubNoToken, map[string]string{"account": g.account, "host": g.host, "why": strings.TrimSpace(done.Stderr)}))
		}
		g.token = strings.TrimSpace(done.Stdout)
	}
	if g.host == "github.com" {
		return []string{"GH_TOKEN=" + g.token}, nil
	}
	return []string{"GH_ENTERPRISE_TOKEN=" + g.token, "GH_HOST=" + g.host}, nil
}

// API calls path with method, sending payload as JSON when it is not nil,
// and reads the answer into into, when GitHub gives one.
func (g *GitHub) API(method, path string, payload, into any) error {
	args := []string{"api"}
	if g.host != "github.com" {
		args = append(args, "--hostname", g.host)
	}
	if method != "GET" {
		args = append(args, "-X", method)
	}
	args = append(args, path)
	var input []byte
	if payload != nil {
		args = append(args, "--input", "-")
		text, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		input = text
	}
	env, err := g.Environment()
	if err != nil {
		return err
	}
	done := g.run(args, input, env)
	words := map[string]string{"method": method, "path": path, "host": g.host}
	switch {
	case done.TimedOut:
		return models.Refused(utils.ResolvePlaceholderString(g.tr.GitHubTooSlow, words))
	case done.Code != 0 && unreachable.MatchString(done.Stderr):
		words["why"] = strings.SplitN(strings.TrimSpace(done.Stderr), "\n", 2)[0]
		return models.Refused(utils.ResolvePlaceholderString(g.tr.GitHubUnreachable, words))
	case done.Code != 0:
		words["why"] = strings.TrimSpace(done.Stderr)
		if words["why"] == "" {
			words["why"] = strings.TrimSpace(done.Stdout)
		}
		return models.Refused(utils.ResolvePlaceholderString(g.tr.GitHubRefused, words))
	case strings.TrimSpace(done.Stdout) == "" || into == nil:
		return nil
	}
	if err := json.Unmarshal([]byte(done.Stdout), into); err != nil {
		words["why"] = err.Error()
		return models.Refused(utils.ResolvePlaceholderString(g.tr.GitHubUnreadable, words))
	}
	return nil
}

// Pull is pull request number of repo, owner/name.
func (g *GitHub) Pull(repo string, number int) (models.GitHubPull, error) {
	var pull models.GitHubPull
	return pull, g.API("GET", "repos/"+repo+"/pulls/"+strconv.Itoa(number), nil, &pull)
}

// Files is each file of the pull request's diff, as GitHub shows it, with
// its patch.
func (g *GitHub) Files(repo string, number int) ([]models.GitHubFile, error) {
	return pages[models.GitHubFile](g, "repos/"+repo+"/pulls/"+strconv.Itoa(number)+"/files")
}

// Reviews is every review sent on the pull request.
func (g *GitHub) Reviews(repo string, number int) ([]models.GitHubReview, error) {
	return pages[models.GitHubReview](g, "repos/"+repo+"/pulls/"+strconv.Itoa(number)+"/reviews")
}

// ReviewComments is every comment of the reviews on the pull request's lines.
func (g *GitHub) ReviewComments(repo string, number int) ([]models.GitHubReviewComment, error) {
	return pages[models.GitHubReviewComment](g, "repos/"+repo+"/pulls/"+strconv.Itoa(number)+"/comments")
}

// CreateReview sends a review, its comments and its event in one request,
// so a failure sends nothing.
func (g *GitHub) CreateReview(repo string, number int, payload any) (models.GitHubReview, error) {
	var review models.GitHubReview
	return review, g.API("POST", "repos/"+repo+"/pulls/"+strconv.Itoa(number)+"/reviews", payload, &review)
}

// Login is the account GitHub answers as.
func (g *GitHub) Login() (string, error) {
	var user models.GitHubUser
	return user.Login, g.API("GET", "user", nil, &user)
}

// OpenPulls is every open pull request of the repository.
func (g *GitHub) OpenPulls(repo string) ([]models.GitHubPull, error) {
	return pages[models.GitHubPull](g, "repos/"+repo+"/pulls?state=open")
}

// Search is every issue and pull request a search finds, such as "is:pr
// is:open author:@me", as GitHub's search gives them, up to the 1,000 it
// gives.
func (g *GitHub) Search(query string) ([]models.GitHubIssue, error) {
	out := []models.GitHubIssue{}
	for page := 1; ; page++ {
		var found models.GitHubSearch
		address := "search/issues?q=" + formEncoded(query) + "&per_page=" + strconv.Itoa(pageSize) + "&page=" + strconv.Itoa(page)
		if err := g.API("GET", address, nil, &found); err != nil {
			return nil, err
		}
		out = append(out, found.Items...)
		if len(found.Items) < pageSize || page == searchPages {
			return out, nil
		}
	}
}

// formEncoded is text as a form encodes a field: letters, digits and
// *-._ as they are, a space as +, and every other byte as %XX.
func formEncoded(text string) string {
	var out strings.Builder
	for _, b := range []byte(text) {
		switch {
		case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9', b == '*', b == '-', b == '.', b == '_':
			out.WriteByte(b)
		case b == ' ':
			out.WriteByte('+')
		default:
			out.WriteString(fmt.Sprintf("%%%02X", b))
		}
	}
	return out.String()
}

func pages[T any](g *GitHub, path string) ([]T, error) {
	join := "?"
	if strings.Contains(path, "?") {
		join = "&"
	}
	out := []T{}
	for page := 1; ; page++ {
		var batch []T
		if err := g.API("GET", path+join+"per_page="+strconv.Itoa(pageSize)+"&page="+strconv.Itoa(page), nil, &batch); err != nil {
			return nil, err
		}
		out = append(out, batch...)
		if len(batch) < pageSize {
			return out, nil
		}
	}
}

// RunGh runs the gh command itself, for a minute at most.
func RunGh(args []string, input []byte, env []string) Ran {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Env = append(os.Environ(), env...)
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	done := Ran{Stdout: stdout.String(), Stderr: stderr.String(), TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded)}
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		done.Code = exit.ExitCode()
	case err != nil:
		done.Code = 1
		if done.Stderr == "" {
			done.Stderr = err.Error()
		}
	}
	return done
}
