package models

import (
	"encoding/json"
	"reflect"
)

// LineEntry is where a comment on one line of a stack's combined diff goes,
// as page.json writes it: [pull, line] for the pull request that changed
// it, with [pull, line, path] when that pull request's own diff names the
// file another way; {"c": [[pull, line], ...]} for an unchanged line, with
// each pull request whose own diff shows it; or {"x": reason} for a line no
// pull request can take.
type LineEntry struct {
	Owner      *LinePlace
	Candidates []LinePlace
	None       string
}

// UnmarshalJSON reads any of the entry's three shapes.
func (e *LineEntry) UnmarshalJSON(data []byte) error {
	var place LinePlace
	if err := json.Unmarshal(data, &place); err == nil {
		e.Owner = &place
		return nil
	}
	var other struct {
		C []LinePlace `json:"c"`
		X *string     `json:"x"`
	}
	if err := json.Unmarshal(data, &other); err != nil {
		return err
	}
	if other.X != nil {
		e.None = *other.X
		return nil
	}
	if other.C == nil {
		return &json.UnmarshalTypeError{Value: "object", Type: reflect.TypeOf(e)}
	}
	e.Candidates = other.C
	return nil
}
