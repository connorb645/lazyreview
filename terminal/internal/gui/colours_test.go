package gui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// The screens drawn in true colour, on a dark terminal and a light one, kept
// with every colour code, so a change to how the app colours a screen shows.
func TestTheColourScreens(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	screens := []struct {
		name string
		draw func(t *testing.T, open func(review string, set ...func(*Options)) App) App
	}{
		{"review-stack", func(t *testing.T, open func(string, ...func(*Options)) App) App {
			return press(t, open(stack), "0", "j")
		}},
		{"diff-split-visual", func(t *testing.T, open func(string, ...func(*Options)) App) App {
			a := typed(t, press(t, open(stack), "space", "f"), "Basket/Basket")
			a = typed(t, press(t, a, "space", "s", ":"), "7")
			return press(t, a, "V", "j")
		}},
		{"diagram-components", func(t *testing.T, open func(string, ...func(*Options)) App) App {
			return press(t, open(stack), "space", "m")
		}},
		{"diagram-types", func(t *testing.T, open func(string, ...func(*Options)) App) App {
			return press(t, open(stack), "space", "m", "enter", "enter")
		}},
		{"diagram-types-filtered", func(t *testing.T, open func(string, ...func(*Options)) App) App {
			return press(t, open(stack), "j", "space", "m", "enter", "enter")
		}},
		{"home", func(t *testing.T, open func(string, ...func(*Options)) App) App {
			return press(t, open("", func(o *Options) { o.Home = "/home/reviewer" }), "j")
		}},
		{"posting-confirm", func(t *testing.T, open func(string, ...func(*Options)) App) App {
			return press(t, open("github.com/octo/widgets/30"), "P", "j", "enter")
		}},
		{"finder", func(t *testing.T, open func(string, ...func(*Options)) App) App {
			a := press(t, open(stack), "space", "f")
			for _, r := range "bask" {
				a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			}
			return a
		}},
		{"keys", func(t *testing.T, open func(string, ...func(*Options)) App) App {
			return press(t, open(stack), "space", "?")
		}},
	}
	for _, s := range screens {
		for _, theme := range []string{"dark", "light"} {
			cmn := commonFor(theme == "dark")
			open := func(review string, set ...func(*Options)) App {
				a, _ := openedIn(t, cmn, &changes{answers: map[string]answer{}}, review, 100, 30, set...)
				return a
			}
			colourScreenIs(t, s.draw(t, open), s.name+"-"+theme)
		}
	}
}

func colourScreenIs(t *testing.T, a App, name string) {
	t.Helper()
	got := a.View() + "\n"
	path := filepath.Join("testdata", "colours", name+".txt")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no colour screen %s yet; write it with go test ./... -update: %v", name, err)
	}
	if string(want) != got {
		t.Errorf("colour screen %s differs.\n--- kept\n%q\n--- now\n%q", name, want, got)
	}
}
