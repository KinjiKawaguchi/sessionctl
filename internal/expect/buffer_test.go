package expect

import (
	"testing"
)

func TestCircularBuffer_WriteAndRead(t *testing.T) {
	buf := NewCircularBuffer(1024)

	data := []byte("hello world")
	n, err := buf.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Fatalf("Write returned %d, want %d", n, len(data))
	}

	got := buf.Bytes()
	if string(got) != "hello world" {
		t.Fatalf("Read got %q, want %q", string(got), "hello world")
	}
}

func TestCircularBuffer_OverwritesOldData(t *testing.T) {
	buf := NewCircularBuffer(8)

	buf.Write([]byte("abcdefgh")) // fill buffer
	buf.Write([]byte("ij"))       // overwrite first 2 bytes

	got := buf.Bytes()
	// After overwrite, buffer should contain the most recent 8 bytes
	if string(got) != "cdefghij" {
		t.Fatalf("Read got %q, want %q", string(got), "cdefghij")
	}
}

func TestCircularBuffer_Clear(t *testing.T) {
	buf := NewCircularBuffer(1024)

	buf.Write([]byte("some data"))
	buf.Clear()

	got := buf.Bytes()
	if len(got) != 0 {
		t.Fatalf("Clear failed: got %q, want empty", string(got))
	}
}

func TestCircularBuffer_MultipleWrites(t *testing.T) {
	buf := NewCircularBuffer(1024)

	buf.Write([]byte("hello "))
	buf.Write([]byte("world"))

	got := buf.Bytes()
	if string(got) != "hello world" {
		t.Fatalf("Read got %q, want %q", string(got), "hello world")
	}
}

func TestCircularBuffer_ConsumeUpTo(t *testing.T) {
	buf := NewCircularBuffer(1024)
	buf.Write([]byte("Username: prompt> "))

	// Consume up to "prompt> " (first 19 bytes = "Username: prompt> ")
	consumed := buf.ConsumeUpTo(18)
	if string(consumed) != "Username: prompt> " {
		t.Fatalf("ConsumeUpTo got %q, want %q", string(consumed), "Username: prompt> ")
	}

	// Remaining should be empty
	got := buf.Bytes()
	if len(got) != 0 {
		t.Fatalf("After consume, got %q, want empty", string(got))
	}
}
