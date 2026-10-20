package testsupport

import (
	"bytes"
	"encoding/json"
)

// Routes are a fake GitHub's answers by route, in the order they were set,
// so they are written as they were given.
type Routes struct {
	keys    []string
	answers map[string]any
}

// Set answers route with answer, keeping the route's place when it is set
// again.
func (r *Routes) Set(route string, answer any) {
	if r.answers == nil {
		r.answers = map[string]any{}
	}
	if _, held := r.answers[route]; !held {
		r.keys = append(r.keys, route)
	}
	r.answers[route] = answer
}

// Raw is every answer as JSON, by route, as FakeGh takes them.
func (r Routes) Raw() (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	for _, k := range r.keys {
		text, err := plainJSON(r.answers[k])
		if err != nil {
			return nil, err
		}
		out[k] = text
	}
	return out, nil
}

// MarshalJSON writes the routes as one object, in their order.
func (r Routes) MarshalJSON() ([]byte, error) {
	var out bytes.Buffer
	out.WriteByte('{')
	for i, k := range r.keys {
		if i > 0 {
			out.WriteByte(',')
		}
		key, err := plainJSON(k)
		if err != nil {
			return nil, err
		}
		answer, err := plainJSON(r.answers[k])
		if err != nil {
			return nil, err
		}
		out.Write(key)
		out.WriteByte(':')
		out.Write(answer)
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

func plainJSON(v any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(out.Bytes(), "\n"), nil
}
