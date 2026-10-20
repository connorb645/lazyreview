package app

import (
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func TestTheAppOpensNoDataHomeButOneNamedByItsFullPath(t *testing.T) {
	want := i18n.EnglishTranslationSet().NoDataHome
	for _, given := range []string{`{}`, `{"dataHome":""}`, `{"dataHome":"reviews"}`} {
		if err := Run("", given, ""); err == nil || err.Error() != want {
			t.Errorf("%s opens as %v", given, err)
		}
	}
}
