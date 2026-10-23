package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/connorb645/lazyreview/internal/ask"
	"github.com/connorb645/lazyreview/internal/cli"
	"github.com/connorb645/lazyreview/internal/config"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/loader"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/review"
	"github.com/connorb645/lazyreview/internal/store"
	"github.com/connorb645/lazyreview/internal/utils"
)

const answersWait = 10 * time.Second

// CommandLine is the lazyreview command. Run does what a command line
// asks, writing each line it says to out as it goes, and returns why it
// could not. `terminal` runs until the terminal app quits; every other
// command returns once done. `config` reads no other setting first, so a
// config with a mistake in it can always be found and fixed.
type CommandLine struct {
	tr       *i18n.TranslationSet
	out      io.Writer
	getenv   func(string) string
	platform string
	home     string
	load     func(tr *i18n.TranslationSet, dataHome string, pulls loader.PullsRequest, branch *loader.BranchRequest) (loader.Loaded, error)
	launch   func(tr *i18n.TranslationSet, given config.UserConfig, readConfig func() (config.Read, error), review string) error
}

// NewCommandLine is the command writing to out, reading the config from
// this process's environment and its user's home folder, loading with Load
// and opening the terminal app with Run.
func NewCommandLine(tr *i18n.TranslationSet, out io.Writer) *CommandLine {
	home, _ := os.UserHomeDir()
	return &CommandLine{tr: tr, out: out, getenv: os.Getenv, platform: runtime.GOOS, home: home, load: Load, launch: Run}
}

// Run does what argv, the command line without the program's name, asks.
func (c *CommandLine) Run(argv []string) error {
	o, err := cli.ParseArguments(c.tr, argv)
	if err != nil {
		return err
	}
	switch o.Command {
	case "help":
		c.say(c.tr.Usage)
		return nil
	case "config":
		return c.config(o)
	}
	given, err := settings(c.tr, c.readConfig, c.getenv)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(given.DataHome, 0o755); err != nil {
		return err
	}
	home := store.DataHome{Root: given.DataHome}
	kept := store.NewReviewStore(c.tr, home, store.NewLocks(c.tr))
	switch o.Command {
	case "load":
		return c.loadReview(given, o)
	case "attach":
		return c.attach(kept, o)
	case "list":
		return c.list(kept)
	case "terminal":
		return c.terminal(given, kept, o)
	case "start":
		c.say(c.tr.CliNoServer)
		return nil
	default:
		return c.stop(home)
	}
}

func (c *CommandLine) loadReview(given config.UserConfig, o cli.Options) error {
	pulls := loader.PullsRequest{Links: o.Positional, Account: o.Account}
	if o.Context || o.NoContext {
		drawn := o.Context
		pulls.Context = &drawn
	}
	if o.Config != nil {
		text, err := c.componentsConfig(*o.Config)
		if err != nil {
			return err
		}
		pulls.Config = text
	}
	var branch *loader.BranchRequest
	if o.Repo != nil {
		branch = &loader.BranchRequest{Repo: *o.Repo, Base: *o.Base, Heads: o.Heads, Title: o.Title, Config: pulls.Config, Context: pulls.Context}
	}
	loaded, err := c.load(c.tr, given.DataHome, pulls, branch)
	if err != nil {
		return err
	}
	counts := loaded.Counts
	c.say(utils.ResolvePlaceholderString(c.tr.CliLoaded, map[string]string{
		"name": c.changeName(loaded.Review), "files": utils.Plural(counts.Files, c.tr.FileNoun), "adds": strconv.Itoa(counts.Adds), "dels": strconv.Itoa(counts.Dels),
		"components": utils.Plural(counts.Components, c.tr.ComponentNoun), "types": utils.Plural(counts.Types, c.tr.TypeNoun),
	}))
	c.say(utils.ResolvePlaceholderString(c.tr.CliOpen, map[string]string{"address": cli.ReviewAddress(loaded.Ref)}))
	c.notes(loaded.Notes)
	return nil
}

func (c *CommandLine) componentsConfig(path string) ([]byte, error) {
	text, err := os.ReadFile(path)
	if err == nil {
		var read any
		err = json.Unmarshal(text, &read)
	}
	if err != nil {
		return nil, models.Refused(utils.ResolvePlaceholderString(c.tr.CliConfigUnreadable, map[string]string{"path": path, "why": err.Error()}))
	}
	return text, nil
}

func (c *CommandLine) attach(kept *store.ReviewStore, o cli.Options) error {
	pull, err := models.ParsePullRef(c.tr, o.Positional[0])
	if err != nil {
		return err
	}
	held, found, err := kept.Find(pull)
	if err != nil {
		return err
	}
	if found == nil {
		return models.Refused(utils.ResolvePlaceholderString(c.tr.CliNotLoaded, map[string]string{"link": o.Positional[0]}))
	}
	text, err := os.ReadFile(o.Positional[1])
	if err != nil {
		return err
	}
	err = kept.Change([]models.ReviewRef{held}, func() ([]store.FileWrite, error) {
		rev, err := kept.Read(held)
		if err != nil {
			return nil, err
		}
		if rev == nil {
			return nil, models.Refused(utils.ResolvePlaceholderString(c.tr.CliNotLoaded, map[string]string{"link": o.Positional[0]}))
		}
		if rev.Moved != nil {
			return nil, models.Refused(utils.ResolvePlaceholderString(c.tr.CliReviewMoved, map[string]string{"to": rev.Moved.To}))
		}
		written, err := review.ReadWrittenReview(c.tr, string(text))
		if err == nil {
			err = review.CheckWrittenReview(c.tr, written, *rev)
		}
		if err != nil {
			return nil, err
		}
		return []store.FileWrite{{Path: kept.Home().Review(held).Written(), Text: string(text)}}, nil
	})
	if err != nil {
		return err
	}
	c.say(utils.ResolvePlaceholderString(c.tr.CliAttached, map[string]string{"id": held.ID(), "address": cli.ReviewAddress(held)}))
	return nil
}

func (c *CommandLine) list(kept *store.ReviewStore) error {
	all, err := kept.List()
	if err != nil {
		return err
	}
	if len(all) == 0 {
		c.say(c.tr.CliNoReviews)
	}
	for _, k := range all {
		if k.Review.Moved != nil {
			c.say(utils.ResolvePlaceholderString(c.tr.CliListedMoved, map[string]string{"id": k.Ref.ID(), "to": k.Review.Moved.To}))
			continue
		}
		c.say(utils.ResolvePlaceholderString(c.tr.CliListed, map[string]string{"address": cli.ReviewAddress(k.Ref), "name": c.changeName(k.Review), "title": k.Review.Title}))
	}
	return nil
}

func (c *CommandLine) terminal(given config.UserConfig, kept *store.ReviewStore, o cli.Options) error {
	id := ""
	if len(o.Positional) > 0 {
		opened, err := c.reviewToOpen(given, kept, o.Positional[0])
		if err != nil {
			return err
		}
		id = opened
	}
	return c.launch(c.tr, given, c.readConfig, id)
}

func (c *CommandLine) reviewToOpen(given config.UserConfig, kept *store.ReviewStore, text string) (string, error) {
	link, err := cli.ReadTerminalLink(c.tr, text)
	if err != nil {
		return "", err
	}
	if link.Review != nil {
		rev, err := kept.Read(link.Review)
		if err != nil {
			return "", err
		}
		if rev == nil {
			return "", models.Refused(utils.ResolvePlaceholderString(c.tr.CliNoReview, map[string]string{"text": link.Review.ID()}))
		}
		return link.Review.ID(), nil
	}
	held, found, err := kept.Find(*link.Pull)
	if err != nil {
		return "", err
	}
	if found != nil {
		return held.ID(), nil
	}
	c.say(utils.ResolvePlaceholderString(c.tr.CliLoading, map[string]string{"url": link.Pull.URL()}))
	loaded, err := c.load(c.tr, given.DataHome, loader.PullsRequest{Links: []string{text}}, nil)
	if err != nil {
		return "", err
	}
	c.notes(loaded.Notes)
	return loaded.Ref.ID(), nil
}

func (c *CommandLine) stop(home store.DataHome) error {
	answers, err := ask.StopAnswers(home, answersWait)
	if err != nil {
		return err
	}
	server, err := NewOldServer(c.tr, home.ServerFile()).Stop()
	if err != nil {
		return err
	}
	if answers == 0 && !server {
		c.say(c.tr.CliNothingRunning)
	}
	if answers > 0 {
		c.say(utils.ResolvePlaceholderString(c.tr.CliAnswersStopped, map[string]string{"answers": utils.Plural(answers, c.tr.AnswerNoun)}))
	}
	if server {
		c.say(c.tr.CliOldServerStopped)
	}
	return nil
}

func (c *CommandLine) config(o cli.Options) error {
	var lines []string
	switch o.Positional[0] {
	case "path":
		file, err := config.FindConfigFile(c.tr, c.getenv, c.platform, c.home, config.IsFile)
		if err != nil {
			return err
		}
		lines = config.PathLines(c.tr, file)
	case "defaults":
		lines = []string{strings.TrimRightFunc(config.DefaultConfigText(c.tr), unicode.IsSpace)}
	case "check":
		read, err := c.readConfig()
		if err != nil {
			return err
		}
		lines = config.CheckLines(c.tr, read)
	default:
		mover := config.NewConfigMove(c.tr, nil)
		plan, err := mover.Plan(c.getenv, c.platform, c.home)
		if err != nil {
			return err
		}
		if o.Check {
			lines = config.MoveCheckLines(c.tr, plan)
		} else if lines, err = mover.Move(plan); err != nil {
			return err
		}
	}
	for _, line := range lines {
		c.say(line)
	}
	return nil
}

func (c *CommandLine) readConfig() (config.Read, error) {
	return config.ReadUserConfig(c.tr, c.getenv, c.platform, c.home)
}

func (c *CommandLine) changeName(r models.Review) string {
	if r.IsBranch() {
		return utils.ResolvePlaceholderString(c.tr.CliBranchName, map[string]string{"head": strings.Join(r.BranchNames(), ", "), "base": r.Base.Ref})
	}
	pulls := make([]string, len(r.Members))
	for i, m := range r.Members {
		pulls[i] = "#" + strconv.Itoa(m.Pull)
	}
	return strings.Join(pulls, ", ")
}

func (c *CommandLine) notes(notes []string) {
	for _, note := range notes {
		c.say(utils.ResolvePlaceholderString(c.tr.CliNote, map[string]string{"note": note}))
	}
}

func (c *CommandLine) say(line string) {
	fmt.Fprintln(c.out, line)
}
