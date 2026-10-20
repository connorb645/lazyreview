package testsupport

import "encoding/json"

// ScriptStep is one step of a recorded script that makes a repository: a
// commit, as [branch, files by path, message], where a file of null is
// removed; a checkout of a branch; or a git command.
type ScriptStep struct {
	Commit   []json.RawMessage `json:"commit"`
	Checkout string            `json:"checkout"`
	Git      []string          `json:"git"`
}
