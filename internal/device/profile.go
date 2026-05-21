package device

import (
	"time"

	"github.com/KinjiKawaguchi/sessionctl/internal/expect"
)

// ProfileEntry is a data-only struct describing a device's CLI behavior.
// No methods with logic — matching is done by the expect package.
type ProfileEntry struct {
	Name           string
	PromptPatterns []expect.Pattern
	LoginPatterns  []expect.Pattern
	ErrorPatterns  []expect.Pattern
	PagerPattern   expect.Pattern
	PagerResponse  string
	CommandTimeout time.Duration
	LoginTimeout   time.Duration
}

// ProfileTable is a flat lookup table for device profiles.
type ProfileTable struct {
	entries []ProfileEntry
	nameIdx map[string]int
}

// NewTable creates an empty profile table.
func NewTable() *ProfileTable {
	return &ProfileTable{
		nameIdx: make(map[string]int),
	}
}

// Register adds a profile to the table.
func (t *ProfileTable) Register(p ProfileEntry) {
	if idx, ok := t.nameIdx[p.Name]; ok {
		t.entries[idx] = p
		return
	}
	t.nameIdx[p.Name] = len(t.entries)
	t.entries = append(t.entries, p)
}

// Lookup retrieves a profile by name.
func (t *ProfileTable) Lookup(name string) (ProfileEntry, bool) {
	idx, ok := t.nameIdx[name]
	if !ok {
		return ProfileEntry{}, false
	}
	return t.entries[idx], true
}

// Names returns all registered profile names.
func (t *ProfileTable) Names() []string {
	names := make([]string, len(t.entries))
	for i, e := range t.entries {
		names[i] = e.Name
	}
	return names
}
