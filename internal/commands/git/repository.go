package git

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

var (
	numstat     = regexp.MustCompile(`(?s)^(-|\d+)\t(-|\d+)\t(.*)$`)
	batchHeader = regexp.MustCompile(`^\S+ (\S+) (\d+)$`)
)

// Repository reads one repository through git. Every command only reads:
// nothing checks out, fetches or writes to the repository or its config.
type Repository struct {
	tr  *i18n.TranslationSet
	dir string
	run Run
}

// NewRepository reads the repository in dir through run.
func NewRepository(tr *i18n.TranslationSet, dir string, run Run) *Repository {
	if run == nil {
		run = RunGit
	}
	return &Repository{tr: tr, dir: dir, run: run}
}

// Resolve is the full commit id ref names.
func (r *Repository) Resolve(ref string) (string, error) {
	out, err := r.text([]string{"rev-parse", "--verify", "--quiet", ref + "^{commit}"}, nil, nil)
	if err != nil {
		return "", models.Refused(utils.ResolvePlaceholderString(r.tr.NotACommit, map[string]string{"ref": ref}))
	}
	return strings.TrimSpace(out), nil
}

// BranchOf is the branch ref names, such as feature/search for HEAD on it,
// and false when it names no branch.
func (r *Repository) BranchOf(ref string) (string, bool, error) {
	out, err := r.text([]string{"rev-parse", "--verify", "--quiet", "--symbolic-full-name", ref}, nil, []int{1})
	name, isBranch := strings.CutPrefix(strings.TrimSpace(out), "refs/heads/")
	return name, isBranch && err == nil, err
}

// SharedFolder is the absolute path of the git folder every worktree of
// the repository shares.
func (r *Repository) SharedFolder() (string, error) {
	out, err := r.text([]string{"rev-parse", "--path-format=absolute", "--git-common-dir"}, nil, nil)
	return strings.TrimSpace(out), err
}

// MergeBase is the best common ancestor of commits a and b.
func (r *Repository) MergeBase(a, b string) (string, error) {
	out, err := r.text([]string{"merge-base", a, b}, nil, nil)
	return strings.TrimSpace(out), err
}

// Diff is git's diff from base to head, with renames found.
func (r *Repository) Diff(base, head string) (string, error) {
	return r.text([]string{"diff", "--no-color", "--no-ext-diff", "--find-renames", base, head}, nil, nil)
}

// Counts is the lines added and removed in each file the diff from base to
// head touches, by its path.
func (r *Repository) Counts(base, head string) (map[string]FileCount, error) {
	out, err := r.text([]string{"diff", "--numstat", "-z", "--find-renames", base, head}, nil, nil)
	if err != nil {
		return nil, err
	}
	counts := map[string]FileCount{}
	fields := strings.Split(out, "\x00")
	for i := 0; i < len(fields); i++ {
		m := numstat.FindStringSubmatch(strings.TrimPrefix(fields[i], "\n"))
		if m == nil {
			continue
		}
		path := m[3]
		if path == "" && i+2 < len(fields) {
			path = fields[i+2]
			i += 2
		}
		adds, _ := strconv.Atoi(m[1])
		dels, _ := strconv.Atoi(m[2])
		counts[path] = FileCount{Adds: adds, Dels: dels, Binary: m[1] == "-"}
	}
	return counts, nil
}

// CommitCount is how many commits head has that base does not.
func (r *Repository) CommitCount(base, head string) (int, error) {
	out, err := r.text([]string{"rev-list", "--count", base + ".." + head}, nil, nil)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(out))
}

// Read is the text of each of paths at commit, nil where the path is
// missing there or is no file.
func (r *Repository) Read(commit string, paths []string) (map[string]*string, error) {
	out := map[string]*string{}
	if len(paths) == 0 {
		return out, nil
	}
	var input strings.Builder
	for _, p := range paths {
		input.WriteString(commit + ":" + p + "\n")
	}
	raw, err := r.text([]string{"cat-file", "--batch"}, []byte(input.String()), nil)
	if err != nil {
		return nil, err
	}
	buffer := []byte(raw)
	at := 0
	for _, path := range paths {
		end := bytes.IndexByte(buffer[at:], '\n')
		if end < 0 {
			out[path] = nil
			continue
		}
		header := string(buffer[at : at+end])
		at += end + 1
		m := batchHeader.FindStringSubmatch(header)
		if m == nil {
			out[path] = nil
			continue
		}
		size, _ := strconv.Atoi(m[2])
		if m[1] == "blob" {
			text := string(buffer[at : at+size])
			out[path] = &text
		} else {
			out[path] = nil
		}
		at += size + 1
	}
	return out, nil
}

// Grep is the lines at commit that match the extended pattern, in the
// files matching globs.
func (r *Repository) Grep(commit, pattern string, globs []string) ([]GrepLine, error) {
	out, err := r.text(append([]string{"grep", "--null", "-I", "-E", "-e", pattern, commit, "--"}, globs...), nil, []int{1})
	if err != nil {
		return nil, err
	}
	var lines []GrepLine
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		cut := strings.IndexByte(line, 0)
		lines = append(lines, GrepLine{Path: line[len(commit)+1 : cut], Text: line[cut+1:]})
	}
	return lines, nil
}

// CountMentions is how many lines in each file at commit name any of
// words, as whole words, in the files matching globs.
func (r *Repository) CountMentions(commit string, words, globs []string) (map[string]int, error) {
	counts := map[string]int{}
	for i := 0; i < len(words); i += 1000 {
		args := []string{"grep", "--null", "-I", "-c", "-w", "-F"}
		for _, w := range words[i:min(len(words), i+1000)] {
			args = append(args, "-e", w)
		}
		out, err := r.text(append(append(args, commit, "--"), globs...), nil, []int{1})
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(out, "\n") {
			if line == "" {
				continue
			}
			cut := strings.IndexByte(line, 0)
			n, _ := strconv.Atoi(line[cut+1:])
			counts[line[len(commit)+1:cut]] += n
		}
	}
	return counts, nil
}

func (r *Repository) text(args []string, input []byte, allowed []int) (string, error) {
	done := r.run(context.Background(), r.dir, append([]string{"-c", "core.quotepath=false"}, args...), nil, input)
	if done.Code != 0 && !slices.Contains(allowed, done.Code) {
		why := strings.SplitN(strings.TrimSpace(done.Stderr), "\n", 2)[0]
		return "", errors.New(utils.ResolvePlaceholderString(r.tr.GitFailed, map[string]string{"command": args[0], "why": why}))
	}
	return done.Stdout, nil
}
