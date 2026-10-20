package app

import (
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func TestTheAppOpensNoDataHomeButOneNamedByItsFullPath(t *testing.T) {
	t.Setenv("LAZYREVIEW_CONFIG_DIR", t.TempDir())
	t.Setenv("LAZYREVIEW_HOME", "")
	t.Setenv("HOME", "")
	if err := Run("", ""); err == nil || err.Error() != i18n.EnglishTranslationSet().NoDataHome {
		t.Errorf("a data home with no home folder to start from opens as %v", err)
	}
}
