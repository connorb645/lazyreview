package home

import (
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func beforeByName(a, b string) bool {
	return utils.LocaleCompare(strings.ToLower(a), strings.ToLower(b)) < 0
}
