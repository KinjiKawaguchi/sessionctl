package session

import (
	"io"
	"sync"
	"time"

	"github.com/KinjiKawaguchi/sessionctl/internal/expect"
)

// SessionEntry holds data for one session. Value type, no methods with logic.
type SessionEntry struct {
	ID         string
	ConnIndex  int
	Depth      int
	Host       string
	ProfileKey string
	CreatedAt  time.Time
	LastUsed   time.Time
}

// ConnectionEntry holds data for one physical connection.
type ConnectionEntry struct {
	Engine    *expect.Engine
	Closer    io.Closer // transport closer (SSH client or TCP conn)
	Depths    int       // current chain depth (number of pushed contexts)
}

// Store is a flat, index-based session and connection store.
// Follows DOD: data in slices, lookup via index maps, no interfaces.
type Store struct {
	mu       sync.RWMutex
	sessions []SessionEntry
	conns    []ConnectionEntry
	idIndex  map[string]int // sessionID → index in sessions slice
}

// NewStore creates an empty session store.
func NewStore() *Store {
	return &Store{
		idIndex: make(map[string]int),
	}
}

// AddConnection appends a connection and returns its index.
func (s *Store) AddConnection(c ConnectionEntry) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := len(s.conns)
	s.conns = append(s.conns, c)
	return idx
}

// GetConnection returns the connection at the given index.
func (s *Store) GetConnection(idx int) ConnectionEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.conns[idx]
}

// AddSession appends a session entry and returns its ID.
func (s *Store) AddSession(e SessionEntry) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	e.LastUsed = e.CreatedAt

	idx := len(s.sessions)
	s.sessions = append(s.sessions, e)
	s.idIndex[e.ID] = idx
	return e.ID
}

// GetSession retrieves a session by ID.
func (s *Store) GetSession(id string) (SessionEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, ok := s.idIndex[id]
	if !ok {
		return SessionEntry{}, false
	}
	return s.sessions[idx], true
}

// ListSessions returns all session IDs.
func (s *Store) ListSessions() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.idIndex))
	for id := range s.idIndex {
		ids = append(ids, id)
	}
	return ids
}

// DeleteSession removes a session by ID. Returns true if found.
func (s *Store) DeleteSession(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.deleteSessionLocked(id)
}

func (s *Store) deleteSessionLocked(id string) bool {
	idx, ok := s.idIndex[id]
	if !ok {
		return false
	}

	delete(s.idIndex, id)

	// Swap-remove from slice to avoid shifting
	last := len(s.sessions) - 1
	if idx != last {
		s.sessions[idx] = s.sessions[last]
		s.idIndex[s.sessions[idx].ID] = idx
	}
	s.sessions = s.sessions[:last]
	return true
}

// PushContext adds a child session on the same connection.
// Sets ConnIndex and increments the connection's Depths.
func (s *Store) PushContext(connIdx int, e SessionEntry) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	e.ConnIndex = connIdx
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	e.LastUsed = e.CreatedAt

	idx := len(s.sessions)
	s.sessions = append(s.sessions, e)
	s.idIndex[e.ID] = idx

	s.conns[connIdx].Depths = e.Depth
	return e.ID
}

// PopContext removes the deepest session on a connection.
// Returns the removed session's ID, or "" if nothing to pop.
func (s *Store) PopContext(connIdx int) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	depth := s.conns[connIdx].Depths
	if depth == 0 {
		return ""
	}

	// Find session at this depth on this connection
	for id, idx := range s.idIndex {
		entry := s.sessions[idx]
		if entry.ConnIndex == connIdx && entry.Depth == depth {
			s.deleteSessionLocked(id)
			s.conns[connIdx].Depths = depth - 1
			return id
		}
	}
	return ""
}

// ActiveSession returns the session ID at the deepest depth on a connection.
func (s *Store) ActiveSession(connIdx int) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	depth := s.conns[connIdx].Depths

	for id, idx := range s.idIndex {
		entry := s.sessions[idx]
		if entry.ConnIndex == connIdx && entry.Depth == depth {
			return id, true
		}
	}
	return "", false
}

// CloseConnection closes the transport, removes all sessions associated
// with a connection, and returns the list of removed session IDs.
func (s *Store) CloseConnection(connIdx int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn := s.conns[connIdx]
	if conn.Engine != nil {
		conn.Engine.Close()
	}
	if conn.Closer != nil {
		conn.Closer.Close()
	}

	var removed []string
	for id, idx := range s.idIndex {
		if s.sessions[idx].ConnIndex == connIdx {
			removed = append(removed, id)
		}
	}

	for _, id := range removed {
		s.deleteSessionLocked(id)
	}
	return removed
}
