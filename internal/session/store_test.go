package session

import (
	"sync"
	"testing"
)

func TestStore_AddAndGet(t *testing.T) {
	s := NewStore()

	connIdx := s.AddConnection(ConnectionEntry{})
	id := s.AddSession(SessionEntry{
		ID:        "sw1",
		ConnIndex: connIdx,
		Depth:     0,
		Host:      "10.1.11.253",
	})

	entry, ok := s.GetSession(id)
	if !ok {
		t.Fatal("session not found")
	}
	if entry.Host != "10.1.11.253" {
		t.Fatalf("host = %q, want %q", entry.Host, "10.1.11.253")
	}
}

func TestStore_GetNotFound(t *testing.T) {
	s := NewStore()

	_, ok := s.GetSession("nonexistent")
	if ok {
		t.Fatal("expected not found, got found")
	}
}

func TestStore_List(t *testing.T) {
	s := NewStore()

	connIdx := s.AddConnection(ConnectionEntry{})
	s.AddSession(SessionEntry{ID: "sw1", ConnIndex: connIdx})
	s.AddSession(SessionEntry{ID: "bravo", ConnIndex: connIdx})

	ids := s.ListSessions()
	if len(ids) != 2 {
		t.Fatalf("len = %d, want 2", len(ids))
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore()

	connIdx := s.AddConnection(ConnectionEntry{})
	s.AddSession(SessionEntry{ID: "sw1", ConnIndex: connIdx})

	ok := s.DeleteSession("sw1")
	if !ok {
		t.Fatal("delete returned false")
	}

	_, found := s.GetSession("sw1")
	if found {
		t.Fatal("session still found after delete")
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := NewStore()
	connIdx := s.AddConnection(ConnectionEntry{})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "sess-" + string(rune('A'+i%26))
			s.AddSession(SessionEntry{ID: id, ConnIndex: connIdx})
			s.GetSession(id)
			s.ListSessions()
		}(i)
	}
	wg.Wait()
	// No race condition = pass
}

func TestStore_PushContext(t *testing.T) {
	s := NewStore()

	connIdx := s.AddConnection(ConnectionEntry{})
	s.AddSession(SessionEntry{
		ID:        "sw1",
		ConnIndex: connIdx,
		Depth:     0,
		Host:      "10.1.11.253",
	})

	childID := s.PushContext(connIdx, SessionEntry{
		ID:    "asa1",
		Host:  "10.1.31.251",
		Depth: 1,
	})

	entry, ok := s.GetSession(childID)
	if !ok {
		t.Fatal("child session not found")
	}
	if entry.Depth != 1 {
		t.Fatalf("depth = %d, want 1", entry.Depth)
	}

	conn := s.GetConnection(connIdx)
	if conn.Depths != 1 {
		t.Fatalf("connection depths = %d, want 1", conn.Depths)
	}
}

func TestStore_PopContext(t *testing.T) {
	s := NewStore()

	connIdx := s.AddConnection(ConnectionEntry{})
	s.AddSession(SessionEntry{
		ID:        "sw1",
		ConnIndex: connIdx,
		Depth:     0,
	})
	s.PushContext(connIdx, SessionEntry{
		ID:    "asa1",
		Depth: 1,
	})

	popped := s.PopContext(connIdx)
	if popped != "asa1" {
		t.Fatalf("popped = %q, want %q", popped, "asa1")
	}

	conn := s.GetConnection(connIdx)
	if conn.Depths != 0 {
		t.Fatalf("connection depths = %d, want 0", conn.Depths)
	}

	// asa1 should be removed
	_, ok := s.GetSession("asa1")
	if ok {
		t.Fatal("asa1 still found after pop")
	}
}

func TestStore_ActiveSession(t *testing.T) {
	s := NewStore()

	connIdx := s.AddConnection(ConnectionEntry{})
	s.AddSession(SessionEntry{ID: "sw1", ConnIndex: connIdx, Depth: 0})
	s.PushContext(connIdx, SessionEntry{ID: "asa1", Depth: 1})

	activeID, ok := s.ActiveSession(connIdx)
	if !ok {
		t.Fatal("no active session")
	}
	if activeID != "asa1" {
		t.Fatalf("active = %q, want %q", activeID, "asa1")
	}
}

func TestStore_CloseConnection(t *testing.T) {
	s := NewStore()

	connIdx := s.AddConnection(ConnectionEntry{})
	s.AddSession(SessionEntry{ID: "sw1", ConnIndex: connIdx, Depth: 0})
	s.PushContext(connIdx, SessionEntry{ID: "asa1", Depth: 1})

	removed := s.CloseConnection(connIdx)
	if len(removed) != 2 {
		t.Fatalf("removed %d sessions, want 2", len(removed))
	}

	_, ok := s.GetSession("sw1")
	if ok {
		t.Fatal("sw1 still found after connection close")
	}
	_, ok = s.GetSession("asa1")
	if ok {
		t.Fatal("asa1 still found after connection close")
	}
}
