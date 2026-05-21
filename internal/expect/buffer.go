package expect

import "sync"

// CircularBuffer is a fixed-size ring buffer that overwrites old data
// when capacity is exceeded. Thread-safe for concurrent read/write.
type CircularBuffer struct {
	data  []byte
	size  int
	start int
	count int
	mu    sync.Mutex
}

// NewCircularBuffer creates a circular buffer with the given capacity.
func NewCircularBuffer(capacity int) *CircularBuffer {
	return &CircularBuffer{
		data: make([]byte, capacity),
		size: capacity,
	}
}

// Write appends data to the buffer, overwriting oldest data if full.
func (b *CircularBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, c := range p {
		pos := (b.start + b.count) % b.size
		b.data[pos] = c
		if b.count == b.size {
			b.start = (b.start + 1) % b.size
		} else {
			b.count++
		}
	}

	return len(p), nil
}

// Bytes returns a copy of the current buffer contents in order.
func (b *CircularBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.bytesLocked()
}

func (b *CircularBuffer) bytesLocked() []byte {
	if b.count == 0 {
		return nil
	}

	result := make([]byte, b.count)
	for i := 0; i < b.count; i++ {
		result[i] = b.data[(b.start+i)%b.size]
	}
	return result
}

// ConsumeUpTo removes and returns the first n bytes from the buffer.
// If n exceeds the current count, all bytes are consumed.
func (b *CircularBuffer) ConsumeUpTo(n int) []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	if n > b.count {
		n = b.count
	}

	result := make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = b.data[(b.start+i)%b.size]
	}
	b.start = (b.start + n) % b.size
	b.count -= n
	return result
}

// Clear resets the buffer to empty.
func (b *CircularBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.start = 0
	b.count = 0
}
