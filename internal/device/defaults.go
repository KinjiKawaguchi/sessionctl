package device

import (
	"regexp"
	"time"

	"github.com/KinjiKawaguchi/sessionctl/internal/expect"
)

// DefaultTable returns a ProfileTable pre-loaded with built-in profiles.
func DefaultTable() *ProfileTable {
	t := NewTable()
	t.Register(ciscoIOSProfile())
	t.Register(yamahaRTXProfile())
	t.Register(unixProfile())
	return t
}

func ciscoIOSProfile() ProfileEntry {
	return ProfileEntry{
		Name: "cisco_ios",
		PromptPatterns: []expect.Pattern{
			{Kind: expect.PatternRegex, Regex: regexp.MustCompile(`[a-zA-Z0-9_-]+[#>]\s*$`)},
		},
		LoginPatterns: []expect.Pattern{
			{Kind: expect.PatternRegex, Regex: regexp.MustCompile(`(?i)username:\s*$`)},
			{Kind: expect.PatternRegex, Regex: regexp.MustCompile(`(?i)password:\s*$`)},
		},
		ErrorPatterns: []expect.Pattern{
			{Kind: expect.PatternExact, Text: "% Invalid"},
			{Kind: expect.PatternExact, Text: "% Ambiguous"},
		},
		PagerPattern:   expect.Pattern{Kind: expect.PatternExact, Text: "--More--"},
		PagerResponse:  " ",
		CommandTimeout: 10 * time.Second,
		LoginTimeout:   30 * time.Second,
	}
}

func yamahaRTXProfile() ProfileEntry {
	return ProfileEntry{
		Name: "yamaha_rtx",
		PromptPatterns: []expect.Pattern{
			{Kind: expect.PatternRegex, Regex: regexp.MustCompile(`[a-zA-Z0-9_-]+[#>]\s*$`)},
		},
		LoginPatterns: []expect.Pattern{
			{Kind: expect.PatternRegex, Regex: regexp.MustCompile(`(?i)username:\s*$`)},
			{Kind: expect.PatternRegex, Regex: regexp.MustCompile(`(?i)password:\s*$`)},
		},
		ErrorPatterns: []expect.Pattern{
			{Kind: expect.PatternExact, Text: "Error:"},
		},
		PagerPattern:   expect.Pattern{Kind: expect.PatternExact, Text: "--More--"},
		PagerResponse:  " ",
		CommandTimeout: 10 * time.Second,
		LoginTimeout:   30 * time.Second,
	}
}

func unixProfile() ProfileEntry {
	return ProfileEntry{
		Name: "unix",
		PromptPatterns: []expect.Pattern{
			{Kind: expect.PatternRegex, Regex: regexp.MustCompile(`[$#%]\s*$`)},
		},
		LoginPatterns: []expect.Pattern{
			{Kind: expect.PatternRegex, Regex: regexp.MustCompile(`(?i)login:\s*$`)},
			{Kind: expect.PatternRegex, Regex: regexp.MustCompile(`(?i)password:\s*$`)},
		},
		ErrorPatterns:  nil,
		PagerPattern:   expect.Pattern{},
		PagerResponse:  "",
		CommandTimeout: 10 * time.Second,
		LoginTimeout:   30 * time.Second,
	}
}
