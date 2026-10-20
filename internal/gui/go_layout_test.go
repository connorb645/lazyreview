package gui

import (
	"fmt"
	"testing"

	"github.com/connorb645/lazyreview/internal/build/layout"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

// The diagram of a recorded review, laid out by the Go layout, reads in
// rows at both sizes: the components, the packages of one, and the types of
// a package, holders above what they hold.
func TestARecordedReviewLaidOutInGoReadsInRows(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {161, 48}} {
		a, _ := opened(t, "github.com/octo/widgets/21", size[0], size[1])
		a.Review.Page.Layout = layout.Of(i18n.EnglishTranslationSet(), a.Review.Page.Model)
		name := fmt.Sprintf("go-layout-%dx%d", size[0], size[1])
		a = showBox(a, types.DiagramPlace{Level: "components", Box: "root"})
		screenIs(t, a, name+"-components")
		a = showBox(a, types.DiagramPlace{Level: "packages", Component: "root", Box: "sources"})
		screenIs(t, a, name+"-packages")
		a = showBox(a, types.DiagramPlace{Level: "types", Component: "root", Package: "sources", Box: "Basket"})
		screenIs(t, a, name+"-types")
	}
}

// The stand-in review, laid out by the Go layout: an area, a group, links
// in layers and types nothing links.
func TestTheStandInLaidOutInGoReadsInRows(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {161, 48}} {
		a := standInApp(size[0], size[1])
		m := a.Review.Page.Model
		m.Areas = []models.Area{{ID: "money", Title: "Money", Components: []string{"core", "ledger"}}}
		m.Summary = []models.ComponentLinks{
			{Source: "core", Target: "ledger", Count: 2, Links: [][2]string{{"Order", "Ledger"}, {"Clock", "Pricer"}}},
			{Source: "shop", Target: "core", Count: 1, Links: [][2]string{{"Store", "Order"}}},
		}
		a.Review.Page.Layout = layout.Of(i18n.EnglishTranslationSet(), m)
		name := fmt.Sprintf("go-layout-stand-in-%dx%d", size[0], size[1])
		a = showBox(a, types.DiagramPlace{Level: "components", Box: "core"})
		screenIs(t, a, name+"-components")
		a = showBox(a, types.DiagramPlace{Level: "types", Component: "core", Package: "model", Box: "Order"})
		screenIs(t, a, name+"-types")
	}
}
