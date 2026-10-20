package utils

import "testing"

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
		if got := FitTitle(c.title, c.width); got != c.want {
			t.Errorf("%q in %d columns is %q, not %q", c.title, c.width, got, c.want)
		}
	}
}
