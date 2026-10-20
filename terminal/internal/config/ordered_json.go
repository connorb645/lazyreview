package config

import (
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var arrayIndex = regexp.MustCompile(`^(?:0|[1-9][0-9]*)$`)

func orderedJSON(text string) (any, error) {
	var check any
	if err := json.Unmarshal([]byte(text), &check); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	return jsonValue(decoder)
}

func jsonValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		out := YAMLMap{Values: map[string]any{}}
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			value, err := jsonValue(decoder)
			if err != nil {
				return nil, err
			}
			name := key.(string)
			if _, held := out.Values[name]; !held {
				out.Keys = append(out.Keys, name)
			}
			out.Values[name] = value
		}
		out.Keys = jsKeyOrder(out.Keys)
		_, err = decoder.Token()
		return out, err
	case json.Delim('['):
		out := []any{}
		for decoder.More() {
			value, err := jsonValue(decoder)
			if err != nil {
				return nil, err
			}
			out = append(out, value)
		}
		_, err = decoder.Token()
		return out, err
	}
	return token, nil
}

func jsKeyOrder(keys []string) []string {
	index := func(key string) (uint64, bool) {
		n, err := strconv.ParseUint(key, 10, 64)
		return n, err == nil && arrayIndex.MatchString(key) && n < 1<<32-1
	}
	var numbered, named []string
	for _, k := range keys {
		if _, ok := index(k); ok {
			numbered = append(numbered, k)
		} else {
			named = append(named, k)
		}
	}
	slices.SortFunc(numbered, func(a, b string) int {
		x, _ := index(a)
		y, _ := index(b)
		return int(x) - int(y)
	})
	return append(numbered, named...)
}
