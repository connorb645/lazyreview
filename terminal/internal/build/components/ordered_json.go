package components

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func orderedJSON(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := orderedValue(decoder)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errors.New("unexpected text after the JSON value")
	}
	return value, nil
}

func orderedValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		object := jsonObject{values: map[string]any{}}
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			value, err := orderedValue(decoder)
			if err != nil {
				return nil, err
			}
			name := key.(string)
			if _, held := object.values[name]; !held {
				object.keys = append(object.keys, name)
			}
			object.values[name] = value
		}
		_, err := decoder.Token()
		return object, err
	case json.Delim('['):
		list := []any{}
		for decoder.More() {
			value, err := orderedValue(decoder)
			if err != nil {
				return nil, err
			}
			list = append(list, value)
		}
		_, err := decoder.Token()
		return list, err
	}
	return token, nil
}
