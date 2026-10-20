package models

import (
	"encoding/json"
	"reflect"
)

// LinePlace is one pull request's own place for a line: its number, its
// line, and its path when it is not the combined diff's.
type LinePlace struct {
	Pull int
	Line int
	Path string
}

// UnmarshalJSON reads [pull, line] or [pull, line, path].
func (p *LinePlace) UnmarshalJSON(data []byte) error {
	var parts []json.RawMessage
	if err := json.Unmarshal(data, &parts); err != nil {
		return err
	}
	if len(parts) < 2 || len(parts) > 3 {
		return &json.UnmarshalTypeError{Value: "array", Type: reflect.TypeOf(p)}
	}
	if err := json.Unmarshal(parts[0], &p.Pull); err != nil {
		return err
	}
	if err := json.Unmarshal(parts[1], &p.Line); err != nil {
		return err
	}
	if len(parts) == 3 {
		return json.Unmarshal(parts[2], &p.Path)
	}
	return nil
}

// MarshalJSON writes [pull, line], or [pull, line, path] when it has a path.
func (p LinePlace) MarshalJSON() ([]byte, error) {
	if p.Path != "" {
		return json.Marshal([]any{p.Pull, p.Line, p.Path})
	}
	return json.Marshal([]int{p.Pull, p.Line})
}
