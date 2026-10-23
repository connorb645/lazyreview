package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/config"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

const asCommand = "LAZYREVIEW_TEST_AS_COMMAND"

func TestMain(m *testing.M) {
	testsupport.ActAsFakeGh()
	if os.Getenv(asCommand) != "" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type ran struct {
	stdout, stderr string
	code           int
}

func lazyreview(t *testing.T, env []string, args ...string) ran {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(self, args...)
	cmd.Env = append(append([]string{}, env...), asCommand+"=1")
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	r := ran{}
	var exit *exec.ExitError
	if err := cmd.Run(); errors.As(err, &exit) {
		r.code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	r.stdout, r.stderr = out.String(), errOut.String()
	return r
}

func standInEnv(home string, more ...string) []string {
	return append([]string{"PATH=" + os.Getenv("PATH"), "HOME=" + home}, more...)
}

type workbench struct {
	repo *testsupport.ThrowawayRepo
	gh   *testsupport.FakeGhCommand
	home string
	env  []string
}

func newWorkbench(t *testing.T) *workbench {
	t.Helper()
	repo := testsupport.NewWorkbench(t)
	origin := repo.Publish(map[int]string{21: "feature/search", 22: "feature/basket", 30: "fix/wording"})
	gh := testsupport.NewFakeGhCommand(t, testsupport.GitHubRoutes(repo, origin, testsupport.WorkbenchPulls, nil))
	home := t.TempDir()
	env := append(standInEnv(os.Getenv("HOME"), "LAZYREVIEW_HOME="+home, "LAZYREVIEW_CONFIG_DIR="+home), gh.Env...)
	return &workbench{repo: repo, gh: gh, home: home, env: env}
}

func link(n int) string {
	return "https://github.com/octo/widgets/pull/" + strconv.Itoa(n)
}

func TestOneCommandLoadsAPullRequestAndPrintsItsAddressWithNoServer(t *testing.T) {
	w := newWorkbench(t)
	loaded := lazyreview(t, w.env, "load", link(30))
	if loaded.code != 0 || loaded.stdout != "loaded #30: 1 file +1 -1, 1 component, 0 types\nopen lazyreview://github.com/octo/widgets/30\n" {
		t.Fatalf("load says %+v", loaded)
	}
	if _, err := os.Stat(filepath.Join(w.home, "server.json")); !os.IsNotExist(err) {
		t.Fatal("something runs in the background")
	}
	stack := lazyreview(t, w.env, "load", link(21), link(22), "--context")
	if stack.code != 0 || !regexp.MustCompile(`^loaded #21, #22: 4 files \+\d+ -\d+, \d+ components?, \d+ types?\nopen lazyreview://github\.com/octo/widgets/21\n$`).MatchString(stack.stdout) {
		t.Fatalf("a stack's load says %+v", stack)
	}

	for _, address := range []string{"github.com/octo/widgets/99", "lazyreview://github.com/octo/widgets/99"} {
		if r := lazyreview(t, w.env, "terminal", address); r.code != 1 || r.stderr != "lazyreview: there is no review github.com/octo/widgets/99; lazyreview list shows every review\n" {
			t.Errorf("terminal %s says %+v", address, r)
		}
	}
	missing := lazyreview(t, w.env, "terminal", link(99))
	if missing.stdout != "loading "+link(99)+"…\n" || !regexp.MustCompile(`^lazyreview: .*pulls/99`).MatchString(missing.stderr) {
		t.Fatalf("a pull request no review holds is not loaded first: %+v", missing)
	}
	if r := lazyreview(t, w.env, "terminal", "widgets"); r.stderr != `lazyreview: "widgets" is not the address of a review or of a pull request; lazyreview list shows every review`+"\n" {
		t.Fatalf("terminal widgets says %+v", r)
	}
	listed := lazyreview(t, w.env, "list")
	if lines := strings.Split(strings.TrimSpace(listed.stdout), "\n"); len(lines) != 2 || !strings.Contains(listed.stdout, "lazyreview://github.com/octo/widgets/21  #21, #22  Stack #21 to #22\n") {
		t.Fatalf("list says %+v", listed)
	}

	file := filepath.Join(w.home, "written.md")
	written := "---\nstack: 21, 22\nhead: " + w.repo.Git("rev-parse", "feature/basket") + "\n---\n## What it does\nAdds search in `Search.swift:4`, then a basket.\n"
	if err := os.WriteFile(file, []byte(written), 0o644); err != nil {
		t.Fatal(err)
	}
	attached := lazyreview(t, w.env, "attach", link(22), file)
	if attached.code != 0 || attached.stdout != "attached to github.com/octo/widgets/21; open lazyreview://github.com/octo/widgets/21\n" {
		t.Fatalf("attach says %+v", attached)
	}
	if kept, _ := os.ReadFile(filepath.Join(w.home, "reviews", "github.com", "octo", "widgets", "21", "written.md")); string(kept) != written {
		t.Fatalf("the written review kept is %q", kept)
	}
	if err := os.WriteFile(file, []byte("---\npull: 30\nhead: 1111111\n---\n## Notes\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale := lazyreview(t, w.env, "attach", link(30), file)
	if stale.code != 1 || !strings.HasPrefix(stale.stderr, "lazyreview: the written review read 1111111, but the review shows head ") {
		t.Fatalf("a written review of an older head says %+v", stale)
	}
}

func TestOneCommandLoadsABranchWithNoPullRequestAndPrintsTheAddressThatOpensIt(t *testing.T) {
	w := newWorkbench(t)
	loaded := lazyreview(t, w.env, "load", "--repo", w.repo.Work, "--base", "main", "--head", "fix/wording", "--title", "Wording")
	lines := strings.Split(loaded.stdout, "\n")
	if loaded.code != 0 || len(lines) != 3 || lines[0] != "loaded fix/wording against main: 1 file +1 -1, 1 component, 0 types" || lines[2] != "" {
		t.Fatalf("a branch's load says %+v", loaded)
	}
	open := lines[1]
	if !regexp.MustCompile(`^open lazyreview://local/work-[0-9a-f]{8}/fix~wording$`).MatchString(open) {
		t.Fatalf("the open line is %q", open)
	}
	address := strings.TrimPrefix(open, "open ")
	record, err := os.ReadFile(filepath.Join(append([]string{w.home, "reviews"}, append(strings.Split(strings.TrimPrefix(address, "lazyreview://"), "/"), "review.json")...)...))
	var review struct{ Title string }
	if err == nil {
		err = json.Unmarshal(record, &review)
	}
	if err != nil || review.Title != "Wording" {
		t.Fatalf("the review's title is %q, %v", review.Title, err)
	}
	again := lazyreview(t, w.env, "load", "--repo", w.repo.Work, "--base", "main", "--head", "fix/wording")
	if strings.Split(again.stdout, "\n")[1] != open {
		t.Fatalf("the same branch loads into another review: %+v", again)
	}
	if listed := lazyreview(t, w.env, "list"); listed.stdout != address+"  fix/wording against main  Wording\n" {
		t.Fatalf("list says %+v", listed)
	}
}

func TestOneCommandLoadsAStackOfBranchesWhereItsBottomBranchAloneLoadedAndAsksGitHubNothing(t *testing.T) {
	w := newWorkbench(t)
	alone := lazyreview(t, w.env, "load", "--repo", w.repo.Work, "--base", "main", "--head", "feature/search")
	stacked := lazyreview(t, w.env, "load", "--repo", w.repo.Work, "--base", "main", "--head", "feature/search", "--head", "feature/basket")
	lines := strings.Split(stacked.stdout, "\n")
	if stacked.code != 0 || len(lines) != 3 || !strings.HasPrefix(lines[0], "loaded feature/search, feature/basket against main: 4 files ") || lines[2] != "" {
		t.Fatalf("a stack of branches' load says %+v", stacked)
	}
	if open := strings.Split(alone.stdout, "\n")[1]; lines[1] != open || !strings.HasSuffix(open, "/feature~search") {
		t.Fatalf("the stack opens at %q, not where its bottom branch alone did, %q", lines[1], open)
	}
	address := strings.TrimPrefix(lines[1], "open ")
	if listed := lazyreview(t, w.env, "list"); listed.stdout != address+"  feature/search, feature/basket against main  Stack feature/search to feature/basket\n" {
		t.Fatalf("list says %+v", listed)
	}
	if calls := w.gh.Calls(); len(calls) != 0 {
		t.Fatalf("gh was called %v", calls)
	}
}

func TestStartSaysNoServerRunsAndStopEndsAServerAnOlderLazyreviewLeftRunning(t *testing.T) {
	root := t.TempDir()
	env := standInEnv(root, "LAZYREVIEW_HOME="+root, "LAZYREVIEW_CONFIG_DIR="+root)
	old := exec.Command("sleep", "60")
	if err := old.Start(); err != nil {
		t.Fatal(err)
	}
	ended := make(chan struct{})
	go func() {
		_ = old.Wait()
		close(ended)
	}()
	defer old.Process.Kill()
	if r := lazyreview(t, env, "start"); r.code != 0 || r.stdout != "lazyreview runs no server now; lazyreview terminal opens the app\n" {
		t.Fatalf("start says %+v", r)
	}
	if r := lazyreview(t, env, "stop"); r.code != 0 || r.stdout != "nothing was running\n" {
		t.Fatalf("stop with nothing running says %+v", r)
	}
	record := filepath.Join(root, "server.json")
	if err := os.WriteFile(record, []byte(`{"pid":`+strconv.Itoa(old.Process.Pid)+`,"address":"http://127.0.0.1:4747","token":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := lazyreview(t, env, "stop"); r.code != 0 || r.stdout != "stopped the server an older lazyreview left running\n" {
		t.Fatalf("stop says %+v", r)
	}
	<-ended
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatal("the old server's record is still there")
	}
	if r := lazyreview(t, env, "stop"); r.stdout != "nothing was running\n" {
		t.Fatalf("stopping again says %+v", r)
	}
}

func TestAFileThatIsNotJSONFailsTheCommandNamingTheFile(t *testing.T) {
	w := newWorkbench(t)
	record := filepath.Join(w.home, "server.json")
	if err := os.WriteFile(record, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if r := lazyreview(t, w.env, "stop"); r.code != 1 || r.stderr != "lazyreview: could not read "+record+": unexpected end of JSON input\n" {
		t.Fatalf("stop with an empty record says %+v", r)
	}
	components := filepath.Join(w.home, "components.json")
	if err := os.WriteFile(components, []byte(`{"components": [`), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := lazyreview(t, w.env, "load", "https://github.com/octo/widgets/pull/1", "--config", components); r.code != 1 || r.stderr != "lazyreview: could not read the config "+components+": unexpected end of JSON input\n" {
		t.Fatalf("a load with a components config that is not JSON says %+v", r)
	}
	if calls := w.gh.Calls(); len(calls) != 0 {
		t.Fatalf("gh was called %v", calls)
	}
}

func TestTheCommandExplainsItselfAndNamesWhatWasWrong(t *testing.T) {
	env := standInEnv(t.TempDir())
	if help := lazyreview(t, env, "help"); help.code != 0 || !strings.HasPrefix(help.stdout, "usage: lazyreview <command> [options]\n") {
		t.Fatalf("help says %+v", help)
	}
	if none := lazyreview(t, env); none.code != 0 || !strings.HasPrefix(none.stdout, "usage: lazyreview <command> [options]\n") {
		t.Fatalf("no command says %+v", none)
	}
	for args, said := range map[string]string{
		"load":  "lazyreview: load needs a pull request, or --repo with --base and --head\n\nusage:",
		"serve": "lazyreview: unknown command \"serve\"\n\nusage:",
	} {
		if r := lazyreview(t, env, args); r.code != 1 || !strings.HasPrefix(r.stderr, said) {
			t.Errorf("%s says %+v", args, r)
		}
	}
}

func configStandIn(t *testing.T) (home, file string, env []string) {
	t.Helper()
	home = t.TempDir()
	folder := filepath.Join(home, "config")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	return home, filepath.Join(folder, "config.yml"), standInEnv(home, "LAZYREVIEW_CONFIG_DIR="+folder)
}

func TestConfigPathSaysWhereTheFileIsAndConfigDefaultsPrintsTheDefaultConfig(t *testing.T) {
	_, file, env := configStandIn(t)
	if r := lazyreview(t, env, "config", "path"); r != (ran{stdout: file + "\nthere is no file there yet, so every setting has its default\n"}) {
		t.Fatalf("config path says %+v", r)
	}
	if err := os.WriteFile(file, []byte("server:\n  port: 4748\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := lazyreview(t, env, "config", "path"); r.stdout != file+"\nit exists\n" {
		t.Fatalf("config path says %+v", r)
	}
	if r := lazyreview(t, env, "config", "defaults"); r != (ran{stdout: config.DefaultConfigText(i18n.EnglishTranslationSet())}) {
		t.Fatalf("config defaults says %+v", r)
	}
}

func TestConfigCheckPrintsEverySettingAsReadAndWhereItCameFrom(t *testing.T) {
	home, file, env := configStandIn(t)
	if err := os.WriteFile(file, []byte("server:\n  port: 4748\ngithub:\n  accounts:\n    - login: lee\nkeybinding:\n  universal:\n    web: gW\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checked := lazyreview(t, append(env, "LAZYREVIEW_MODEL=claude-sonnet"), "config", "check")
	unused := "\n\n# server does nothing now, so it can go: lazyreview runs no server now; lazyreview terminal opens the app\n" +
		"# keybinding.universal.web does nothing now, so it can go: lazyreview has no web app; keybinding.reviewView.drawings opens a written review's drawings in the browser\n"
	if checked.code != 0 || !strings.HasSuffix(checked.stdout, unused) {
		t.Fatalf("config check says %+v", checked)
	}
	head, theme, _ := strings.Cut(strings.TrimSuffix(checked.stdout, unused)+"\n", "\ngui:\n")
	want := strings.Join([]string{
		"the config at " + file + ", on top of the defaults:",
		"",
		padded("dataHome: "+filepath.Join(home, ".lazyreview"), 38) + " # default",
		"",
		"claude:",
		"  command: claude                      # default",
		"  model: claude-sonnet                 # LAZYREVIEW_MODEL",
		"",
		"os:",
		"  editor: ''                           # default",
		"",
		"github:",
		"  accounts:                            # the file",
		"    - login: lee",
		"      teams: []",
		"  hiddenRepositories: []               # default",
		"",
	}, "\n")
	if head != want {
		t.Fatalf("config check begins:\n%s\nwant:\n%s", head, want)
	}
	everyDefault := regexp.MustCompile(`^ *\w+:( +# [A-Za-z ]+)?$|# default$`)
	for _, line := range strings.Split(strings.TrimRight(theme, "\n"), "\n") {
		if line != "" && !everyDefault.MatchString(line) {
			t.Errorf("a colour or a key is not its default: %q", line)
		}
	}
	if !regexp.MustCompile(`(?m)^    dark:\n      defaultFgColor: '#f1f0fb' +# default\n`).MatchString(theme) {
		t.Fatalf("the theme reads:\n%s", theme)
	}
}

func padded(text string, width int) string {
	return text + strings.Repeat(" ", max(0, width-len(text)))
}

func TestAConfigWithAMistakeStopsACommandNamingTheFileAndWhatIsWrong(t *testing.T) {
	_, file, env := configStandIn(t)
	if err := os.WriteFile(file, []byte("gui:\n  lineNumbers: sideways\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := lazyreview(t, env, "list"); r.code != 1 || !strings.HasPrefix(r.stderr, "lazyreview: "+file+` gives gui.lineNumbers "sideways", `) {
		t.Fatalf("list with a config that has a mistake says %+v", r)
	}
	if r := lazyreview(t, env, "config", "check"); r.code != 1 {
		t.Fatalf("config check with a mistake says %+v", r)
	}
	for _, args := range [][]string{{"help"}, {"config", "path"}, {"config", "defaults"}} {
		if r := lazyreview(t, env, args...); r.code != 0 {
			t.Errorf("%v with a config that has a mistake says %+v", args, r)
		}
	}
}

func TestConfigMoveMovesSettingsJSONAndTheVariablesIntoTheConfigAfterACheckThatWritesNothing(t *testing.T) {
	home, file, env := configStandIn(t)
	data := filepath.Join(home, ".lazyreview")
	if err := os.Mkdir(data, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "settings.json"), []byte(`{"accounts": [{"login": "lee"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	env = append(env, "LAZYREVIEW_MODEL=claude-sonnet")
	checked := lazyreview(t, env, "config", "move", "--check")
	if checked.code != 0 || !regexp.MustCompile(`^would write .*config\.yml:\n(?s:.*)\nnothing was written; lazyreview config move makes the move\n$`).MatchString(checked.stdout) {
		t.Fatalf("config move --check says %+v", checked)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("the check wrote the config")
	}
	if moved := lazyreview(t, env, "config", "move"); moved.code != 0 {
		t.Fatalf("config move says %+v", moved)
	}
	want := "# yaml-language-server: $schema=https://raw.githubusercontent.com/connorb645/lazyreview/main/schema/config.json\n# lazyreview's config. docs/Config.md lists every setting and its default.\nclaude:\n  model: claude-sonnet\ngithub:\n  accounts:\n    - login: lee\n"
	if text, err := os.ReadFile(file); err != nil || string(text) != want {
		t.Fatalf("the config moved is %q, %v", text, err)
	}
	if _, err := os.Stat(filepath.Join(data, "settings.json.moved")); err != nil {
		t.Fatal("settings.json was not kept as settings.json.moved")
	}
	if r := lazyreview(t, env, "config", "move"); r.code != 1 || !strings.HasSuffix(r.stderr, "exists already, and the move never overwrites a config\n") {
		t.Fatalf("a second move says %+v", r)
	}
}
