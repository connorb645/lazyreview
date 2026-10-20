package components

import (
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type checker struct {
	tr       *i18n.TranslationSet
	problems []string
}

func (c *checker) say(word string, values map[string]string) {
	c.problems = append(c.problems, utils.ResolvePlaceholderString(word, values))
}

func (c *checker) keys(item any, allowed []string, label string) (jsonObject, bool) {
	object, ok := item.(jsonObject)
	if !ok {
		c.say(c.tr.ComponentsNotAnObject, map[string]string{"label": label})
		return object, false
	}
	for _, k := range object.keys {
		if !slices.Contains(allowed, k) {
			c.say(c.tr.ComponentsUnknownKey, map[string]string{"label": label, "key": k})
		}
	}
	return object, true
}

func (c *checker) text(item jsonObject, key, label string, required bool) string {
	v, held := item.values[key]
	if !held {
		if required {
			c.say(c.tr.ComponentsNeeds, map[string]string{"label": label, "key": key})
		}
		return ""
	}
	s, ok := v.(string)
	if !ok || strings.TrimFunc(s, isJSSpace) == "" {
		c.say(c.tr.ComponentsNotText, map[string]string{"label": label, "key": key})
		return ""
	}
	return s
}

func (c *checker) id(item jsonObject, label string, seen map[string]bool) string {
	c.text(item, "id", label, true)
	id, ok := item.values["id"].(string)
	if !ok {
		return ""
	}
	if !idPattern.MatchString(id) {
		c.say(c.tr.ComponentsBadID, map[string]string{"label": label})
	}
	if seen[id] {
		c.say(c.tr.ComponentsIDTwice, map[string]string{"label": label, "id": id})
	}
	seen[id] = true
	return id
}

func (c *checker) list(item jsonObject, key, label string) []any {
	v, held := item.values[key]
	if !held {
		return nil
	}
	list, ok := v.([]any)
	if !ok {
		c.say(c.tr.ComponentsNotAList, map[string]string{"label": label, "key": key})
		return nil
	}
	return list
}

func (c *checker) strings(item jsonObject, key, label string) []string {
	list := c.list(item, key, label)
	out := []string{}
	bad := false
	for _, v := range list {
		s, ok := v.(string)
		bad = bad || !ok || s == ""
		out = append(out, jsText(v))
	}
	if bad {
		c.say(c.tr.ComponentsNotStrings, map[string]string{"label": label, "key": key})
	}
	return out
}

func (c *checker) named(kind string, item any, number int) string {
	if object, ok := item.(jsonObject); ok {
		if id, ok := object.values["id"].(string); ok {
			return utils.ResolvePlaceholderString(c.tr.ComponentsNamed, map[string]string{"kind": kind, "id": id})
		}
	}
	return utils.ResolvePlaceholderString(c.tr.ComponentsNumbered, map[string]string{"kind": kind, "number": strconv.Itoa(number)})
}

func isJSSpace(r rune) bool {
	return unicode.IsSpace(r) || r == '\uFEFF'
}

func jsText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
		return x.String()
	case bool:
		return strconv.FormatBool(x)
	case nil:
		return "null"
	case []any:
		parts := make([]string, len(x))
		for i, item := range x {
			if item != nil {
				parts[i] = jsText(item)
			}
		}
		return strings.Join(parts, ",")
	}
	return "[object Object]"
}
