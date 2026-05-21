package expect

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

const defaultPollInterval = 50 * time.Millisecond

// ErrDisconnected is returned when the underlying connection has closed.
var ErrDisconnected = fmt.Errorf("session disconnected")

// Engine performs expect/send operations on an I/O stream.
// A background goroutine continuously reads from the reader into a buffer.
// Expect polls the buffer for pattern matches.
// Send writes directly to the writer.
type Engine struct {
	writer       io.Writer
	buffer       *CircularBuffer
	mu           sync.Mutex
	done         chan struct{}
	closed       bool
	disconnected chan struct{} // closed when readLoop exits (EOF or error)
}

// NewEngine creates an expect engine.
// It starts a background goroutine that reads from reader into a circular buffer.
func NewEngine(reader io.Reader, writer io.Writer, bufSize int) *Engine {
	e := &Engine{
		writer:       writer,
		buffer:       NewCircularBuffer(bufSize),
		done:         make(chan struct{}),
		disconnected: make(chan struct{}),
	}
	go e.readLoop(reader)
	return e
}

// readLoop continuously reads from the reader into the buffer.
// Closes e.disconnected when the reader returns an error (EOF or connection drop).
func (e *Engine) readLoop(reader io.Reader) {
	defer close(e.disconnected)
	buf := make([]byte, 4096)
	for {
		select {
		case <-e.done:
			return
		default:
		}

		n, err := reader.Read(buf)
		if n > 0 {
			e.buffer.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

// Disconnected returns a channel that is closed when the connection drops.
func (e *Engine) Disconnected() <-chan struct{} {
	return e.disconnected
}

// Send writes data to the underlying writer.
// Returns ErrDisconnected if the connection has dropped.
func (e *Engine) Send(_ context.Context, data string) error {
	select {
	case <-e.disconnected:
		return ErrDisconnected
	default:
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	_, err := e.writer.Write([]byte(data))
	return err
}

// Expect waits until the pattern matches data in the buffer, or the context expires.
// Returns all buffered data up to and including the match.
func (e *Engine) Expect(ctx context.Context, pattern Pattern) ([]byte, error) {
	output, _, err := e.ExpectAny(ctx, []Pattern{pattern})
	return output, err
}

// ExpectAny waits for any of the patterns to match.
// Returns the matched data, the index of the matched pattern, and any error.
func (e *Engine) ExpectAny(ctx context.Context, patterns []Pattern) ([]byte, int, error) {
	ticker := time.NewTicker(defaultPollInterval)
	defer ticker.Stop()

	for {
		data := e.buffer.Bytes()
		if len(data) > 0 {
			idx, end, ok := MatchAny(patterns, data)
			if ok {
				consumed := e.buffer.ConsumeUpTo(end)
				return consumed, idx, nil
			}
		}

		select {
		case <-e.disconnected:
			// Drain any remaining data in buffer before returning error
			data = e.buffer.Bytes()
			if len(data) > 0 {
				if idx, end, ok := MatchAny(patterns, data); ok {
					consumed := e.buffer.ConsumeUpTo(end)
					return consumed, idx, nil
				}
			}
			return nil, 0, fmt.Errorf("expect: %w", ErrDisconnected)
		case <-ctx.Done():
			return nil, 0, fmt.Errorf("expect: %w", ctx.Err())
		case <-ticker.C:
			// poll again
		}
	}
}

// Pager defines a pager pattern and the response to send when matched.
type Pager struct {
	Pattern  Pattern
	Response string
}

// ExpectWithPager waits for any of the target patterns while automatically
// handling pager prompts (e.g., "--More--"). When the pager pattern matches,
// the response is sent and matching continues.
func (e *Engine) ExpectWithPager(ctx context.Context, targets []Pattern, pager Pager) ([]byte, int, error) {
	ticker := time.NewTicker(defaultPollInterval)
	defer ticker.Stop()

	var accumulated []byte
	allPatterns := append(targets, pager.Pattern)

	for {
		data := e.buffer.Bytes()
		if len(data) > 0 {
			idx, end, ok := MatchAny(allPatterns, data)
			if ok {
				consumed := e.buffer.ConsumeUpTo(end)
				accumulated = append(accumulated, consumed...)

				if idx < len(targets) {
					return accumulated, idx, nil
				}
				// Pager matched: send response and continue
				e.writer.Write([]byte(pager.Response))
				continue
			}
		}

		select {
		case <-ctx.Done():
			return accumulated, 0, fmt.Errorf("expect: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

// Buffer returns the underlying circular buffer for direct inspection.
func (e *Engine) Buffer() *CircularBuffer {
	return e.buffer
}

// Close stops the background reader goroutine.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.closed {
		e.closed = true
		close(e.done)
	}
	return nil
}
