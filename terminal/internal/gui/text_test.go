package gui

import (
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func TestAPathAmongWordsLosesItsFirstFoldersAndTheWordsStay(t *testing.T) {
	at := &models.Lines{Path: "Sources/Store/Web/BasketFlow/Pricing/PricingRule.swift", Side: "RIGHT", Line: 3}
	if got := presentation.ItemWords(testCommon().Tr, presentation.Item{Kind: "link", Place: at}, 40); got != "…/Pricing/PricingRule.swift line 3" {
		t.Errorf("a link's place in 40 columns is %q", got)
	}
	if got := presentation.ItemWords(testCommon().Tr, presentation.Item{Kind: "link", Place: at}, 80); got != "Sources/Store/Web/BasketFlow/Pricing/PricingRule.swift line 3" {
		t.Errorf("a link's place that fits is %q", got)
	}
	if got := utils.FitPathIn("Diagram · Store › ", "Store/Web/Basket Flow/Pricing", "", 3); got != "Diagram · Store › Store/Web/Basket Flow/Pricing" {
		t.Errorf("with no room for the path, the line is left whole for its frame to cut: %q", got)
	}
}
