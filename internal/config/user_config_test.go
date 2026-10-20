package config

import (
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
)

func TestTheHandOverIsReadStrictly(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	got, err := Decode(`{"path":"/home/lee/.config/lazyreview/config.yml","editor":"nvim","theme":{"dark":{"accentColor":"#a796ff","pullRequestColors":["#3987e5"],"componentColors":{"blue":"#7fa8f8"}}}}`, tr)
	if err != nil || got.Path != "/home/lee/.config/lazyreview/config.yml" || got.Editor != "nvim" {
		t.Fatalf("the hand-over reads as %+v, %v", got, err)
	}
	if dark := got.Theme.Dark; dark.AccentColor != "#a796ff" || dark.PullRequestColors[0] != "#3987e5" || dark.ComponentColors["blue"] != "#7fa8f8" {
		t.Fatalf("the theme reads as %+v", dark)
	}
	for _, wrong := range []string{``, `{"path":"x","keys":{}}`, `{"path":3}`, `{"path":"x"} {}`, `[]`, `{"theme":{"dark":{"accent":"#fff"}}}`} {
		if _, err := Decode(wrong, tr); err == nil {
			t.Errorf("%q was read", wrong)
		}
	}
}

func TestTheHandOverGivesTheKeysOfEachPlaceByAction(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	got, err := Decode(`{"keybinding":{"universal":{"quit":["Z Z","ctrl+c"],"refresh":[]},"home":{"jumpToRepository":["1","","3"]}}}`, tr)
	if err != nil {
		t.Fatalf("the keys were not read: %v", err)
	}
	if k := got.Keybinding; len(k.Universal.Quit) != 2 || k.Universal.Quit[0] != "Z Z" || k.Universal.Refresh == nil || len(k.Universal.Refresh) != 0 || k.Home.JumpToRepository[1] != "" {
		t.Fatalf("the keys read as %+v", k)
	}
	for _, wrong := range []string{`{"keybinding":{"universal":{"jump":["j"]}}}`, `{"keybinding":{"everywhere":{}}}`, `{"keybinding":{"universal":{"quit":"Z Z"}}}`} {
		if _, err := Decode(wrong, tr); err == nil {
			t.Errorf("%q was read", wrong)
		}
	}
}
