package expect

import (
	"regexp"
	"testing"
)

func TestExactPattern_MatchFound(t *testing.T) {
	p := Pattern{Kind: PatternExact, Text: "Password:"}
	data := []byte("Enter Password: ")

	end, ok := Match(p, data)
	if !ok {
		t.Fatal("expected match, got no match")
	}
	if end != 15 { // "Enter Password:" ends at index 15
		t.Fatalf("end = %d, want 15", end)
	}
}

func TestExactPattern_NoMatch(t *testing.T) {
	p := Pattern{Kind: PatternExact, Text: "Password:"}
	data := []byte("Username: admin")

	_, ok := Match(p, data)
	if ok {
		t.Fatal("expected no match, got match")
	}
}

func TestRegexPattern_MatchFound(t *testing.T) {
	p := Pattern{
		Kind:  PatternRegex,
		Regex: regexp.MustCompile(`[a-zA-Z0-9_-]+[#>]\s*$`),
	}
	data := []byte("Switch#")

	end, ok := Match(p, data)
	if !ok {
		t.Fatal("expected match, got no match")
	}
	if end != 7 {
		t.Fatalf("end = %d, want 7", end)
	}
}

func TestRegexPattern_NoMatch(t *testing.T) {
	p := Pattern{
		Kind:  PatternRegex,
		Regex: regexp.MustCompile(`[a-zA-Z0-9_-]+[#>]\s*$`),
	}
	data := []byte("some random output\n")

	_, ok := Match(p, data)
	if ok {
		t.Fatal("expected no match, got match")
	}
}

func TestMatchAny_ReturnsFirstMatch(t *testing.T) {
	patterns := []Pattern{
		{Kind: PatternExact, Text: "Username:"},
		{Kind: PatternExact, Text: "Password:"},
	}
	data := []byte("Enter Password: ")

	idx, end, ok := MatchAny(patterns, data)
	if !ok {
		t.Fatal("expected match, got no match")
	}
	if idx != 1 {
		t.Fatalf("matched index = %d, want 1", idx)
	}
	if end != 15 {
		t.Fatalf("end = %d, want 15", end)
	}
}

func TestMatchAny_NoMatch(t *testing.T) {
	patterns := []Pattern{
		{Kind: PatternExact, Text: "Username:"},
		{Kind: PatternExact, Text: "Password:"},
	}
	data := []byte("Welcome to the system\n")

	_, _, ok := MatchAny(patterns, data)
	if ok {
		t.Fatal("expected no match, got match")
	}
}
