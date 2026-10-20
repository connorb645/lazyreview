package testsupport

import "encoding/json"

// Call is one call gh was asked to make: its method, its path, the JSON it
// was sent and the token it was given.
type Call struct {
	Method string
	Path   string
	Body   json.RawMessage
	Token  string
}
