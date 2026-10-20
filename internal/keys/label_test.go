package keys

import "testing"

func TestALabelReadsAsTheHelpWritesKeys(t *testing.T) {
	for sequence, want := range map[string]string{
		"g g": "gg", "] p": "]p", "Z Z": "ZZ", "z a": "za", "j": "j", "G": "G", ":": ":",
		"space ?": "Space ?", "space v": "Space v", "space": "Space",
		"enter": "Enter", "esc": "Esc", "ctrl+o": "Ctrl-o", "ctrl+d": "Ctrl-d", "tab": "Ctrl-i", "down": "Down",
	} {
		if got := Label(sequence); got != want {
			t.Errorf("%q is labelled %q, not %q", sequence, got, want)
		}
	}
}
