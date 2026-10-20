package gui

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/gui/context"
)

const notFetched = "could not fetch #22's head ffffffff from octo/widgets: git fetch failed: fatal: git upload-pack: not our ref ffffffffffffffffffffffffffffffffffffffff"

func TestARefusedReloadSaysSoAndKeepsTheWholeReasonInAPopup(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {161, 48}} {
		a, _, changed := openedWith(t, stack, size[0], size[1])
		refused, _ := json.Marshal(map[string]string{"error": notFetched})
		changed.set("POST /api/reviews/"+stack+"/refresh {}", answer{Status: 422, Body: refused})
		a = press(t, a, "R")
		if a.Notice != "Nothing was loaded; Esc closes why." || !a.Bad || !a.Has(context.MenuKey) {
			t.Fatalf("at %dx%d a refused reload says %q", size[0], size[1], a.Notice)
		}
		if len(a.Menu.Rows) != 1 || a.Menu.Title != "Load the review again" || a.Menu.Rows[0].Right != "C"+notFetched[1:] || !strings.Contains(a.View(), strings.Repeat("f", 40)) {
			t.Fatalf("at %dx%d the reason is not shown whole:\n%s", size[0], size[1], a.View())
		}
		screenIs(t, a, fmt.Sprintf("reload-refused-%dx%d", size[0], size[1]))
		if a = press(t, a, "esc"); a.Has(context.MenuKey) || a.Review == nil {
			t.Fatalf("Esc closes the reason and leaves the review open")
		}
	}
}

func TestALoadThatFailsSaysSoAndKeepsTheWholeReasonInAPopup(t *testing.T) {
	a, _, _ := homeOn(t, "", 80, 24)
	a = press(t, a, "enter")
	if a.Screen() != context.ScreenHome || a.Home.Loading != "" || !a.Bad || a.Notice != "Nothing was loaded; Esc closes why." {
		t.Fatalf("a failed load: screen %d, %q", a.Screen(), a.Notice)
	}
	if want := "GitHub refused GET repos/octo/gadgets/pulls/7: HTTP 404: Not Found (fake gh has no GET repos/octo/gadgets/pulls/7)"; a.Menu.Title != "Load octo/gadgets #7" || len(a.Menu.Rows) != 1 || a.Menu.Rows[0].Right != want {
		t.Fatalf("the reason is not shown whole:\n%s", a.View())
	}
	screenIs(t, a, "home-load-refused-80x24")
}
