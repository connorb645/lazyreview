package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

type recordedCommands struct {
	Checked []struct {
		Text    string            `json:"text"`
		Env     map[string]string `json:"env"`
		Checked string            `json:"checked"`
	} `json:"checked"`
	Dumps []json.RawMessage `json:"dumps"`
	Moved []struct {
		Values json.RawMessage `json:"values"`
		Text   string          `json:"text"`
	} `json:"moved"`
	Moves []struct {
		Platform string            `json:"platform"`
		Env      map[string]string `json:"env"`
		Files    map[string]string `json:"files"`
		Failures map[string][]int  `json:"failures"`
		Error    string            `json:"error"`
		Plan     *struct {
			ConfigPath     string          `json:"configPath"`
			ConfigExists   bool            `json:"configExists"`
			SettingsPath   string          `json:"settingsPath"`
			SettingsExists bool            `json:"settingsExists"`
			Values         json.RawMessage `json:"values"`
			From           []MoveSource    `json:"from"`
			Variables      []string        `json:"variables"`
		} `json:"plan"`
		Check     []string          `json:"check"`
		Moved     []string          `json:"moved"`
		MoveError string            `json:"moveError"`
		After     map[string]string `json:"after"`
	} `json:"moves"`
}

func recordedCommandAnswers(t *testing.T) recordedCommands {
	t.Helper()
	var r recordedCommands
	text, err := os.ReadFile(testsupport.Testdata("config-commands.json"))
	if err == nil {
		err = json.Unmarshal(text, &r)
	}
	if err != nil {
		t.Fatalf("could not read the recorded config commands: %v", err)
	}
	return r
}

func decoded(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	v, err := orderedJSON(string(raw))
	if err != nil {
		t.Fatalf("%s: %v", raw, err)
	}
	return v
}

func getenvOf(env map[string]string) func(string) string {
	return func(name string) string { return env[name] }
}

// Each recorded config is written by config check in Go as Node wrote it,
// to the byte, with the source of each value beside it.
func TestEachConfigIsCheckedAsNodeCheckedIt(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, c := range recordedCommandAnswers(t).Checked {
		var read Read
		var err error
		if c.Env == nil {
			read, err = ConfigFrom(tr, c.Text, recordedPath, recordedHome)
		} else {
			folder := t.TempDir()
			if err := os.WriteFile(filepath.Join(folder, "config.yml"), []byte(c.Text), 0o644); err != nil {
				t.Fatal(err)
			}
			env := map[string]string{"LAZYREVIEW_CONFIG_DIR": folder}
			for k, v := range c.Env {
				env[k] = v
			}
			read, err = ReadUserConfig(tr, getenvOf(env), "linux", recordedHome)
		}
		if err != nil {
			t.Errorf("%q was refused: %v", c.Text, err)
			continue
		}
		if got := CheckedText(tr, read); got != c.Checked {
			t.Errorf("%q %v is checked as\n%s\nnot\n%s", c.Text, c.Env, got, c.Checked)
		}
	}
}

// Each recorded value is written as YAML in Go as js-yaml wrote it, to the
// byte: every printable character first, inside and last in a string, the
// words another type reads, white space, other alphabets, characters YAML
// escapes, several lines, keys, and lists and maps at every depth.
func TestEachValueIsWrittenAsJSYAMLWroteIt(t *testing.T) {
	answers := recordedCommandAnswers(t)
	if len(answers.Dumps) < 4000 {
		t.Fatalf("only %d recorded values", len(answers.Dumps))
	}
	for _, raw := range answers.Dumps {
		var pair []json.RawMessage
		var want string
		if err := json.Unmarshal(raw, &pair); err != nil || len(pair) != 2 || json.Unmarshal(pair[1], &want) != nil {
			t.Fatalf("%s is not a value and its YAML", raw)
		}
		if got := dumpYAML(decoded(t, pair[0])); got != want {
			t.Errorf("%s is written as\n%q\nnot\n%q", pair[0], got, want)
		}
	}
	for _, m := range answers.Moved {
		values, _ := decoded(t, m.Values).(YAMLMap)
		if got := MovedText(i18n.EnglishTranslationSet(), values); got != m.Text {
			t.Errorf("%s is moved as\n%s\nnot\n%s", m.Values, got, m.Text)
		}
	}
}

type memoryMoveFiles struct {
	files    map[string]string
	failures map[string][]int
	calls    map[string]int
}

func (f *memoryMoveFiles) step(name string) error {
	f.calls[name]++
	if slices.Contains(f.failures[name], f.calls[name]) {
		return errors.New("the disk refused the " + name)
	}
	return nil
}

func (f *memoryMoveFiles) Exists(path string) bool {
	_, held := f.files[path]
	return held
}

func (f *memoryMoveFiles) Read(path string) (string, error) {
	return f.files[path], nil
}

func (f *memoryMoveFiles) Write(path, text string) error {
	if err := f.step("write"); err != nil {
		return err
	}
	if f.Exists(path) {
		return errors.New(path + " exists")
	}
	f.files[path] = text
	return nil
}

func (f *memoryMoveFiles) Rename(from, to string) error {
	if err := f.step("rename"); err != nil {
		return err
	}
	f.files[to] = f.files[from]
	delete(f.files, from)
	return nil
}

func (f *memoryMoveFiles) Remove(path string) error {
	if err := f.step("remove"); err != nil {
		return err
	}
	delete(f.files, path)
	return nil
}

func (f *memoryMoveFiles) Folder(string) error {
	return f.step("folder")
}

// Each recorded move plans, checks and moves in Go as in Node: the same
// plan, the same lines, the same refusal, and the same files after it,
// including a move that fails partway and puts back what it did, or says
// what it could not. Only a settings.json that is not JSON may say why in
// other words.
func TestEachMoveIsMadeAsNodeMadeIt(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, c := range recordedCommandAnswers(t).Moves {
		files := &memoryMoveFiles{files: map[string]string{}, failures: c.Failures, calls: map[string]int{}}
		for path, text := range c.Files {
			files.files[path] = text
		}
		move := NewConfigMove(tr, files)
		plan, err := move.Plan(getenvOf(c.Env), c.Platform, recordedHome)
		if c.Error != "" {
			unreadable := strings.Contains(c.Error, " could not be read: ")
			switch {
			case err == nil:
				t.Errorf("%v %v planned, where Node refused: %s", c.Env, c.Files, c.Error)
			case unreadable && !strings.HasPrefix(err.Error(), strings.SplitAfter(c.Error, " could not be read: ")[0]):
				t.Errorf("%v %v was refused as %q, not as unreadable", c.Env, c.Files, err)
			case !unreadable && err.Error() != c.Error:
				t.Errorf("%v %v was refused as\n%s\nnot\n%s", c.Env, c.Files, err, c.Error)
			}
			continue
		}
		if err != nil {
			t.Errorf("%v %v was refused: %v", c.Env, c.Files, err)
			continue
		}
		want := c.Plan
		if plan.ConfigPath != want.ConfigPath || plan.ConfigExists != want.ConfigExists || plan.SettingsPath != want.SettingsPath || plan.SettingsExists != want.SettingsExists ||
			Shown(plan.Values) != Shown(decoded(t, want.Values)) || !reflect.DeepEqual(plan.From, want.From) || !reflect.DeepEqual(plan.Variables, want.Variables) {
			t.Errorf("%v %v plans\n%+v\nnot\n%+v %s", c.Env, c.Files, plan, *want, want.Values)
			continue
		}
		if got := MoveCheckLines(tr, plan); !reflect.DeepEqual(got, c.Check) {
			t.Errorf("%v %v checks as\n%s\nnot\n%s", c.Env, c.Files, strings.Join(got, "\n"), strings.Join(c.Check, "\n"))
		}
		said, err := move.Move(plan)
		switch {
		case c.MoveError != "" && (err == nil || err.Error() != c.MoveError):
			t.Errorf("%v %v %v moved with %v, not the refusal %q", c.Env, c.Files, c.Failures, err, c.MoveError)
		case c.MoveError == "" && (err != nil || !reflect.DeepEqual(said, c.Moved)):
			t.Errorf("%v %v moved saying\n%s (%v)\nnot\n%s", c.Env, c.Files, strings.Join(said, "\n"), err, strings.Join(c.Moved, "\n"))
		}
		after := map[string]string{}
		for path, text := range files.files {
			after[strings.Replace(path, "."+strconv.Itoa(os.Getpid())+".moving", ".<pid>.moving", 1)] = text
		}
		if !reflect.DeepEqual(after, c.After) {
			t.Errorf("%v %v %v leaves\n%v\nnot\n%v", c.Env, c.Files, c.Failures, after, c.After)
		}
	}
}
