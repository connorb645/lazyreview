package ask

import "regexp"

type outward struct {
	pattern *regexp.Regexp
	kind    string
}
