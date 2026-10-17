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
		draw func(t *testing.T, light func(*Options)) App
	}{
		{"review-stack", func(t *testing.T, light func(*Options)) App {
			a, _ := opened(t, stack, 100, 30, light)
			return press(t, a, "0", "j")
		}},
		{"diff-split-visual", func(t *testing.T, light func(*Options)) App {
			a, _ := opened(t, stack, 100, 30, light)
			a = typed(t, press(t, a, "space", "f"), "Basket/Basket")
			a = typed(t, press(t, a, "space", "s", ":"), "7")
			return press(t, a, "V", "j")
		}},
		{"diagram-components", func(t *testing.T, light func(*Options)) App {
			a, _ := opened(t, stack, 100, 30, light)
			return press(t, a, "space", "m")
		}},
		{"diagram-types", func(t *testing.T, light func(*Options)) App {
			a, _ := opened(t, stack, 100, 30, light)
			return press(t, a, "space", "m", "enter", "enter")
		}},
		{"home", func(t *testing.T, light func(*Options)) App {
			a, _ := openedOn(t, &changes{answers: map[string]answer{}}, "", 100, 30, light, func(o *Options) { o.Home = "/home/reviewer" })
			return press(t, a, "j")
		}},
		{"posting-confirm", func(t *testing.T, light func(*Options)) App {
			a, _ := opened(t, "github.com/octo/widgets/30", 100, 30, light)
			return press(t, a, "P", "j", "enter")
		}},
		{"finder", func(t *testing.T, light func(*Options)) App {
			a, _ := opened(t, stack, 100, 30, light)
			a = press(t, a, "space", "f")
			for _, r := range "bask" {
				a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			}
			return a
		}},
		{"keys", func(t *testing.T, light func(*Options)) App {
			a, _ := opened(t, stack, 100, 30, light)
			return press(t, a, "space", "?")
		}},
	}
	for _, s := range screens {
		for _, theme := range []string{"dark", "light"} {
			light := func(o *Options) { o.Dark = theme == "dark" }
			colourScreenIs(t, s.draw(t, light), s.name+"-"+theme)
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
