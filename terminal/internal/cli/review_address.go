package cli

import (
	"fmt"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// ReviewAddress is the address of the review ref names, as load, attach
// and list print it and terminal opens it: lazyreview://<review id>, each
// part of the id percent-encoded as a link's part is.
func ReviewAddress(ref models.ReviewRef) string {
	parts := strings.Split(ref.ID(), "/")
	for i, p := range parts {
		parts[i] = encodedPart(p)
	}
	return "lazyreview://" + strings.Join(parts, "/")
}

func encodedPart(part string) string {
	var out strings.Builder
	for _, b := range []byte(part) {
		if b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || strings.IndexByte("-_.!~*'()", b) >= 0 {
			out.WriteByte(b)
		} else {
			fmt.Fprintf(&out, "%%%02X", b)
		}
	}
	return out.String()
}
