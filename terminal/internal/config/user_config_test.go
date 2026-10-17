package config

import (
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func TestTheHandOverIsReadStrictly(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	got, err := Decode(`{"path":"/home/lee/.config/lazyreview/config.yml","editor":"nvim"}`, tr)
	if err != nil || got != (UserConfig{Path: "/home/lee/.config/lazyreview/config.yml", Editor: "nvim"}) {
		t.Fatalf("the hand-over reads as %+v, %v", got, err)
	}
	for _, wrong := range []string{``, `{"path":"x","keys":{}}`, `{"path":3}`, `{"path":"x"} {}`, `[]`} {
		if _, err := Decode(wrong, tr); err == nil {
			t.Errorf("%q was read", wrong)
		}
	}
}
