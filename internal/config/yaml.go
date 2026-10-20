package config

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

var (
	decimal  = regexp.MustCompile(`^[-+]?[0-9]+$`)
	hex      = regexp.MustCompile(`^[-+]?0x[0-9a-fA-F]+$`)
	octal    = regexp.MustCompile(`^[-+]?0o[0-7]+$`)
	binary   = regexp.MustCompile(`^[-+]?0b[01]+$`)
	floating = regexp.MustCompile(`^(?:[-+]?[0-9]+(?:\.[0-9]*)?(?:[eE][-+]?[0-9]+)?|\.[0-9]+(?:[eE][-+]?[0-9]+)?)$`)
)

// ParseYAML is the value text holds, read as lazyreview has always read
// YAML, with the core schema: a date such as 2026-10-01 stays text, a key
// written twice is refused, and a mapping keeps its keys' order. A mapping
// is a YAMLMap, a list []any, a number float64, and a scalar otherwise a
// string, a bool or nil. Empty text holds nil. When the text cannot be
// read, the error says why and where, as "<reason> (<line>:<column>)".
func ParseYAML(tr *i18n.TranslationSet, text string) (any, error) {
	file, err := parser.ParseBytes([]byte(text), 0, parser.AllowDuplicateMapKey())
	if err != nil {
		var syntax *yaml.SyntaxError
		if errors.As(err, &syntax) && syntax.Token != nil {
			return nil, yamlError(tr, syntax.Message, syntax.Token.Position)
		}
		return nil, err
	}
	if len(file.Docs) == 0 || file.Docs[0].Body == nil {
		return nil, nil
	}
	r := reader{tr: tr, anchors: map[string]any{}}
	return r.value(file.Docs[0].Body)
}

func (r reader) value(node ast.Node) (any, error) {
	switch n := node.(type) {
	case nil:
		return nil, nil
	case *ast.DocumentNode:
		return r.value(n.Body)
	case *ast.MappingNode:
		return r.mapping(n.Values)
	case *ast.MappingValueNode:
		return r.mapping([]*ast.MappingValueNode{n})
	case *ast.SequenceNode:
		out := []any{}
		for _, v := range n.Values {
			item, err := r.value(v)
			if err != nil {
				return nil, err
			}
			out = append(out, item)
		}
		return out, nil
	case *ast.AnchorNode:
		v, err := r.value(n.Value)
		if err == nil {
			r.anchors[n.Name.GetToken().Value] = v
		}
		return v, err
	case *ast.AliasNode:
		return r.anchors[n.Value.GetToken().Value], nil
	case *ast.TagNode:
		return r.tagged(n)
	case *ast.LiteralNode:
		return n.Value.Value, nil
	case *ast.CommentGroupNode:
		return nil, nil
	}
	tok := node.GetToken()
	if tok == nil {
		return nil, nil
	}
	if tok.Type == token.SingleQuoteType || tok.Type == token.DoubleQuoteType {
		if s, ok := node.(*ast.StringNode); ok {
			return s.Value, nil
		}
		return tok.Value, nil
	}
	return resolve(tok.Value), nil
}

func (r reader) mapping(entries []*ast.MappingValueNode) (any, error) {
	out := YAMLMap{Values: map[string]any{}}
	for _, entry := range entries {
		var keyNode ast.Node = entry.Key
		if explicit, ok := keyNode.(*ast.MappingKeyNode); ok {
			keyNode = explicit.Value
		}
		key, err := r.key(keyNode)
		if err != nil {
			return nil, err
		}
		if _, held := out.Values[key]; held {
			return nil, yamlError(r.tr, r.tr.ConfigDuplicatedKey, keyNode.GetToken().Position)
		}
		v, err := r.value(entry.Value)
		if err != nil {
			return nil, err
		}
		out.Keys = append(out.Keys, key)
		out.Values[key] = v
	}
	return out, nil
}

func (r reader) key(node ast.Node) (string, error) {
	if _, ok := node.(*ast.MergeKeyNode); ok {
		return "<<", nil
	}
	v, err := r.value(node)
	if err != nil {
		return "", err
	}
	return jsText(v), nil
}

func (r reader) tagged(n *ast.TagNode) (any, error) {
	text := ""
	if tok := n.Value.GetToken(); tok != nil {
		text = tok.Value
	}
	switch n.Start.Value {
	case "!!str":
		return text, nil
	case "!!map", "!!seq":
		return r.value(n.Value)
	case "!!null":
		return nil, nil
	case "!!bool", "!!int", "!!float":
		v := resolve(text)
		switch v.(type) {
		case bool:
			if n.Start.Value == "!!bool" {
				return v, nil
			}
		case float64:
			if n.Start.Value != "!!bool" {
				return v, nil
			}
		}
	}
	return nil, yamlError(r.tr, "unknown tag "+n.Start.Value, n.Start.Position)
}

// resolve is a plain scalar's value by the core schema.
func resolve(text string) any {
	switch text {
	case "", "~", "null", "Null", "NULL":
		return nil
	case "true", "True", "TRUE":
		return true
	case "false", "False", "FALSE":
		return false
	case ".inf", ".Inf", ".INF", "+.inf", "+.Inf", "+.INF":
		return math.Inf(1)
	case "-.inf", "-.Inf", "-.INF":
		return math.Inf(-1)
	case ".nan", ".NaN", ".NAN":
		return math.NaN()
	}
	for _, based := range []struct {
		pattern *regexp.Regexp
		base    int
	}{{hex, 16}, {octal, 8}, {binary, 2}} {
		if based.pattern.MatchString(text) {
			sign, digits := 1.0, strings.TrimLeft(text, "+-")
			if strings.HasPrefix(text, "-") {
				sign = -1
			}
			n, err := strconv.ParseUint(digits[2:], based.base, 64)
			if err == nil {
				return sign * float64(n)
			}
		}
	}
	if decimal.MatchString(text) || floating.MatchString(text) {
		if f, err := strconv.ParseFloat(text, 64); err == nil {
			return f
		}
	}
	return text
}

func yamlError(tr *i18n.TranslationSet, reason string, at *token.Position) error {
	return errors.New(utils.ResolvePlaceholderString(tr.ConfigYAMLError, map[string]string{
		"reason": strings.TrimSpace(reason), "line": strconv.Itoa(at.Line), "column": strconv.Itoa(at.Column),
	}))
}
