package models

import "encoding/json"

// HeadingPart is a piece of the line under a change's title.
type HeadingPart struct {
	Text string
	Bold bool
}

// UnmarshalJSON reads a part as the server writes it, [text, bold].
func (p *HeadingPart) UnmarshalJSON(data []byte) error {
	var pair [2]json.RawMessage
	if err := json.Unmarshal(data, &pair); err != nil {
		return err
	}
	if err := json.Unmarshal(pair[0], &p.Text); err != nil {
		return err
	}
	return json.Unmarshal(pair[1], &p.Bold)
}

// MarshalJSON writes the part as [text, bold].
func (p HeadingPart) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{p.Text, p.Bold})
}
