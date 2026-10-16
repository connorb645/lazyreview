// Package keys reads Vim-style key sequences: a count, then one key or a few,
// such as "5j", "gg", "]c" or a leader key and the key after it.
package keys

import (
	"sort"
	"strconv"
	"strings"
)

// Reader holds the keys pressed so far toward a sequence. Read returns the
// next Reader, so reading a key changes nothing else.
type Reader struct {
	count   string
	pending []string
}

// Read takes one key, named as Bindings name keys, and returns the next
// Reader and what the key did. A digit starts or adds to a count, unless
// the bindings take that digit itself; "0" only adds to a count already
// started.
func (r Reader) Read(key string, bindings Bindings) (Reader, Result) {
	if len(r.pending) == 0 && isDigit(key) && (r.count != "" || (key != "0" && bindings[key] == "")) {
		return Reader{count: r.count + key}, Result{Waiting: true}
	}
	sequence := strings.Join(append(append([]string{}, r.pending...), key), " ")
	if action, ok := bindings[sequence]; ok {
		count, counted := 1, r.count != ""
		if counted {
			count, _ = strconv.Atoi(r.count)
		}
		return Reader{}, Result{Action: action, Count: count, Counted: counted}
	}
	if bindings.begins(sequence) {
		return Reader{count: r.count, pending: append(append([]string{}, r.pending...), key)}, Result{Waiting: true}
	}
	return Reader{}, Result{}
}

// Typed is what has been typed toward a sequence so far, such as "5g", for
// the status line to show.
func (r Reader) Typed() string {
	return r.count + strings.Join(r.pending, "")
}

// Pending is the keys typed so far after any count, joined by spaces, as
// Bindings name them.
func (r Reader) Pending() string {
	return strings.Join(r.pending, " ")
}

// Bindings maps a sequence, its keys joined by spaces, such as "g g" or
// "space l", to the action it does.
type Bindings map[string]string

// With returns these bindings with more added, the added ones winning.
func (b Bindings) With(more Bindings) Bindings {
	out := Bindings{}
	for k, v := range b {
		out[k] = v
	}
	for k, v := range more {
		out[k] = v
	}
	return out
}

// After lists the keys that can follow a sequence, each with its action,
// sorted by key, such as what can follow the leader.
func (b Bindings) After(sequence string) [][2]string {
	var out [][2]string
	for k, v := range b {
		if strings.HasPrefix(k, sequence+" ") && !strings.Contains(k[len(sequence)+1:], " ") {
			out = append(out, [2]string{k[len(sequence)+1:], v})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

func (b Bindings) begins(sequence string) bool {
	for k := range b {
		if strings.HasPrefix(k, sequence+" ") {
			return true
		}
	}
	return false
}

func isDigit(key string) bool {
	return len(key) == 1 && key[0] >= '0' && key[0] <= '9'
}
