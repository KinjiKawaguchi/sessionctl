package device

import (
	"testing"

	"github.com/KinjiKawaguchi/sessionctl/internal/expect"
)

func TestCiscoIOS_PromptPatterns(t *testing.T) {
	p, ok := DefaultTable().Lookup("cisco_ios")
	if !ok {
		t.Fatal("cisco_ios profile not found")
	}

	tests := []struct {
		input string
		want  bool
	}{
		{"Switch#", true},
		{"Switch>", true},
		{"Cat3560-CX#", true},
		{"ciscoasa# ", true},
		{"some output\n", false},
	}

	for _, tt := range tests {
		matched := false
		for _, pat := range p.PromptPatterns {
			if _, ok := expect.Match(pat, []byte(tt.input)); ok {
				matched = true
				break
			}
		}
		if matched != tt.want {
			t.Errorf("prompt match %q = %v, want %v", tt.input, matched, tt.want)
		}
	}
}

func TestCiscoIOS_LoginPatterns(t *testing.T) {
	p, _ := DefaultTable().Lookup("cisco_ios")

	tests := []struct {
		input string
		want  bool
	}{
		{"Username: ", true},
		{"Password: ", true},
		{"random text", false},
	}

	for _, tt := range tests {
		matched := false
		for _, pat := range p.LoginPatterns {
			if _, ok := expect.Match(pat, []byte(tt.input)); ok {
				matched = true
				break
			}
		}
		if matched != tt.want {
			t.Errorf("login match %q = %v, want %v", tt.input, matched, tt.want)
		}
	}
}

func TestCiscoIOS_PagerPattern(t *testing.T) {
	p, _ := DefaultTable().Lookup("cisco_ios")

	_, ok := expect.Match(p.PagerPattern, []byte(" --More-- "))
	if !ok {
		t.Fatal("pager pattern should match '--More--'")
	}
}

func TestYamahaRTX_PromptPatterns(t *testing.T) {
	p, ok := DefaultTable().Lookup("yamaha_rtx")
	if !ok {
		t.Fatal("yamaha_rtx profile not found")
	}

	tests := []struct {
		input string
		want  bool
	}{
		{"RTX1210> ", true},
		{"RTX1200# ", true},
		{"random\n", false},
	}

	for _, tt := range tests {
		matched := false
		for _, pat := range p.PromptPatterns {
			if _, ok := expect.Match(pat, []byte(tt.input)); ok {
				matched = true
				break
			}
		}
		if matched != tt.want {
			t.Errorf("prompt match %q = %v, want %v", tt.input, matched, tt.want)
		}
	}
}

func TestYamahaRTX_LoginPatterns(t *testing.T) {
	p, _ := DefaultTable().Lookup("yamaha_rtx")

	// Yamaha has two-stage login: Username then Password
	tests := []struct {
		input string
		want  bool
	}{
		{"Username: ", true},
		{"Password: ", true},
	}

	for _, tt := range tests {
		matched := false
		for _, pat := range p.LoginPatterns {
			if _, ok := expect.Match(pat, []byte(tt.input)); ok {
				matched = true
				break
			}
		}
		if matched != tt.want {
			t.Errorf("login match %q = %v, want %v", tt.input, matched, tt.want)
		}
	}
}

func TestUnix_PromptPatterns(t *testing.T) {
	p, ok := DefaultTable().Lookup("unix")
	if !ok {
		t.Fatal("unix profile not found")
	}

	tests := []struct {
		input string
		want  bool
	}{
		{"user@bravo:~$ ", true},
		{"root@victor:~# ", true},
		{"% ", true}, // FreeBSD csh
		{"random text\n", false},
	}

	for _, tt := range tests {
		matched := false
		for _, pat := range p.PromptPatterns {
			if _, ok := expect.Match(pat, []byte(tt.input)); ok {
				matched = true
				break
			}
		}
		if matched != tt.want {
			t.Errorf("prompt match %q = %v, want %v", tt.input, matched, tt.want)
		}
	}
}

func TestTable_LookupNotFound(t *testing.T) {
	_, ok := DefaultTable().Lookup("nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestTable_Register(t *testing.T) {
	tbl := DefaultTable()

	tbl.Register(ProfileEntry{
		Name: "custom_device",
		PromptPatterns: []expect.Pattern{
			{Kind: expect.PatternExact, Text: "CUSTOM>"},
		},
	})

	_, ok := tbl.Lookup("custom_device")
	if !ok {
		t.Fatal("custom_device not found after register")
	}
}
