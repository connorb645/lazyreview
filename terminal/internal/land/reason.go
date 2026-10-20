package land

import (
	"regexp"
	"strings"
)

var (
	rejectedRef = regexp.MustCompile(`(?m)^!\t[^\t]*\t\[([^\]]*)\](?: \((.*)\))?$`)
	remoteSaid  = regexp.MustCompile(`(?m)^remote: (.*\S)`)
	gitPrefix   = regexp.MustCompile(`^(?:fatal|error): `)
)

func reason(ran Ran) string {
	rejected := rejectedRef.FindStringSubmatch(ran.Out)
	if rejected == nil {
		first := strings.SplitN(ran.Err, "\n", 2)[0]
		if first == "" {
			first = failedFallback
		}
		return plain(first)
	}
	why := rejected[2]
	if why == "" {
		why = rejected[1]
	}
	if said := remoteSaid.FindStringSubmatch(ran.Err); said != nil {
		why += ", saying: " + plain(said[1])
	}
	return why
}

func plain(line string) string {
	return gitPrefix.ReplaceAllString(line, "")
}

func list(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
