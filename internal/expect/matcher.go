package expect

import (
	"bytes"
	"regexp"
)

// PatternKind identifies the type of pattern matching to perform.
type PatternKind uint8

const (
	PatternExact PatternKind = iota
	PatternRegex
)

// Pattern is a value type representing a match rule.
// Uses a tagged union instead of an interface to avoid virtual dispatch.
type Pattern struct {
	Kind  PatternKind
	Text  string         // used when Kind == PatternExact
	Regex *regexp.Regexp // used when Kind == PatternRegex
}

// Match checks if the pattern matches anywhere in data.
// Returns the end position of the match and whether it matched.
func Match(p Pattern, data []byte) (end int, ok bool) {
	switch p.Kind {
	case PatternExact:
		idx := bytes.Index(data, []byte(p.Text))
		if idx < 0 {
			return 0, false
		}
		return idx + len(p.Text), true

	case PatternRegex:
		loc := p.Regex.FindIndex(data)
		if loc == nil {
			return 0, false
		}
		return loc[1], true

	default:
		return 0, false
	}
}

// MatchAny checks each pattern in order and returns the index of the first
// pattern that matches, along with the end position.
func MatchAny(patterns []Pattern, data []byte) (idx int, end int, ok bool) {
	for i, p := range patterns {
		if e, matched := Match(p, data); matched {
			return i, e, true
		}
	}
	return 0, 0, false
}
