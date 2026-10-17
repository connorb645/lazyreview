package app

import (
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/server"
)

func TestATitleThatIsAPathLosesItsFirstFoldersAndAnyOtherTitleItsEnd(t *testing.T) {
	cases := []struct {
		title string
		width int
		want  string
	}{
		{"docs/store/help/local", 30, "docs/store/help/local"},
		{"Store/Web/Basket Flow/Pricing", 29, "Store/Web/Basket Flow/Pricing"},
		{"Store/Web/Basket Flow/Pricing", 28, "…/Web/Basket Flow/Pricing"},
		{"Store/Web/Basket Flow/Pricing", 22, "…/Basket Flow/Pricing"},
		{"Store/Web/Basket Flow/Pricing", 12, "…/Pricing"},
		{"Store/Web/Basket Flow/Pricing", 6, "…icing"},
		{"Sources/Basket/", 12, "…/Basket/"},
		{"Pricing rules for every basket", 20, "Pricing rules for e…"},
		{"Store/Web/Basket Flow/Pricing", 0, ""},
	}
	for _, c := range cases {
		if got := fitTitle(c.title, c.width); got != c.want {
			t.Errorf("%q in %d columns is %q, not %q", c.title, c.width, got, c.want)
		}
	}
}

func TestAPathAmongWordsLosesItsFirstFoldersAndTheWordsStay(t *testing.T) {
	at := &server.Lines{Path: "Sources/Store/Web/BasketFlow/Pricing/PricingRule.swift", Side: "RIGHT", Line: 3}
	if got := itemWords(item{kind: "link", place: at}, 40); got != "…/Pricing/PricingRule.swift line 3" {
		t.Errorf("a link's place in 40 columns is %q", got)
	}
	if got := itemWords(item{kind: "link", place: at}, 80); got != "Sources/Store/Web/BasketFlow/Pricing/PricingRule.swift line 3" {
		t.Errorf("a link's place that fits is %q", got)
	}
	if got := fitPathIn("Diagram · Store › ", "Store/Web/Basket Flow/Pricing", "", 3); got != "Diagram · Store › Store/Web/Basket Flow/Pricing" {
		t.Errorf("with no room for the path, the line is left whole for its frame to cut: %q", got)
	}
}
