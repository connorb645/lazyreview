package app

import (
	"io"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/loader"
)

func TestTheAppOpensNoDataHomeButOneNamedByItsFullPath(t *testing.T) {
	t.Setenv("LAZYREVIEW_CONFIG_DIR", t.TempDir())
	t.Setenv("LAZYREVIEW_HOME", "")
	t.Setenv("HOME", "")
	if err := Run(""); err == nil || err.Error() != i18n.EnglishTranslationSet().NoDataHome {
		t.Errorf("a data home with no home folder to start from opens as %v", err)
	}
	if err := Load(io.Discard, loader.PullsRequest{}, nil); err == nil || err.Error() != i18n.EnglishTranslationSet().NoDataHome {
		t.Errorf("a load into a data home with no home folder to start from loads as %v", err)
	}
}
