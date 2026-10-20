package helpers

import (
	"testing"

	"github.com/connorb645/lazyreview/internal/gui/context"
)

func TestManyRepositoriesFoldToTheirTitlesAndTheListFollowsTheKeys(t *testing.T) {
	wants := make([]int, 12)
	for i := range wants {
		wants[i] = 5
	}
	l := layHome(80, 24, 4, context.NormalScreen, wants)
	var heights []int
	for _, r := range l.Panels {
		heights = append(heights, r.H)
	}
	if heights[4] != 12 || heights[0] != 1 || heights[11] != 1 || l.Panels[11].Y != 22 {
		t.Fatalf("eleven panels fold to one row and the twelfth takes the rest: %v", heights)
	}
	many := make([]int, 30)
	for i := range many {
		many[i] = 5
	}
	l = layHome(80, 24, 20, context.NormalScreen, many)
	shown := 0
	for i, r := range l.Panels {
		if r.H > 0 {
			shown++
			if i == 20 && r.H != 3 {
				t.Fatalf("the panel with the keys keeps a row: %d", r.H)
			}
		}
	}
	if shown != 21 || l.Panels[0].H != 0 || l.Panels[20].H == 0 {
		t.Fatalf("with more panels than rows, the ones around the keys show: %d shown", shown)
	}
}
