// Package player implements backend side stream buffering.
package player

import (
	"io"
	"sync"

	"github.com/pkg/errors"
)

// Error constant for player package.
type Error string

func (e Error) String() string {
	return "err: " + string(e)
}

func (e Error) Error() string {
	return string(e)
}

// Possible Error values.
const (
	// ErrMiss indicates that ID is out ouf range.
	ErrMiss Error = "buffer miss"
	// ErrBufferTooSmall indicates invalid length of provided buffer.
	ErrBufferTooSmall Error = "buffer is too small"
	// ErrTooBigWrite indicates attempt to write too large data chunk.
	ErrTooLargeWrite Error = "write is too large"
	// ErrEmpty means that Buffer is empty.
	ErrEmpty Error = "buffer is empty"
)

// Buffer represents in-memory buffer for stream.
type Buffer struct {
	segment  int64
	maxCount int64
	count    int64
	lastID   int64
	firstID  int64
	overflow bool
	l        sync.Mutex
	data     []byte
}

// Config is configuration for Buffer.
type Config struct {
	Segment int64
	Count   int64
	Start   int64
	// AllowOverflow allows Buffer.Write to accept buffer which size
	// is larger than maximum internal buffer size (count * segment).
	AllowOverflow bool
}

// New creates new Buffer with specified settings. If value in Config is zero,
// sensible defaults will be used.
func New(cfg Config) *Buffer {
	if cfg.Segment == 0 {
		cfg.Segment = 1024
	}
	if cfg.Count == 0 {
		cfg.Count = 8
	}
	return &Buffer{
		segment:  cfg.Segment,
		maxCount: cfg.Count,
		firstID:  cfg.Start,
		overflow: cfg.AllowOverflow,
	}
}

// NewDefault returns new buffer with default config.
func NewDefault() *Buffer {
	return New(Config{})
}

// SetCount sets maximum segment count.
func (b *Buffer) SetCount(count int64) {
	b.l.Lock()
	b.maxCount = count
	b.l.Unlock()
}

// SegmentSize returns size of segment.
func (b Buffer) SegmentSize() int64 {
	return b.segment
}

// Count returns current maximum segment count.
func (b *Buffer) Count() int64 {
	b.l.Lock()
	defer b.l.Unlock()
	return b.maxCount
}

// Size returns current data length.
func (b *Buffer) Size() int {
	b.l.Lock()
	defer b.l.Unlock()
	return len(b.data)
}

// getSegment returns buffer for segment with id. No checks and locks.
func (b *Buffer) getSegment(id int64) []byte {
	start := b.segment * (id - b.firstID)
	return b.data[start : b.segment+start]
}

// Write appends internal buffer with new data.
func (b *Buffer) Write(buf []byte) (int, error) {
	if int64(len(buf)) > b.segment*b.maxCount {
		// buffer length is bigger than maximum size.
		if !b.overflow {
			return 0, errors.Wrap(ErrTooLargeWrite, "failed to write")
		}
	}

	b.l.Lock()
	b.data = append(b.data, buf...) // ALLOCATIONS: suboptimal.
	b.count = int64(len(b.data)) / b.segment

	// updating buffer window (firstID and lastID)
	b.lastID = b.count + b.firstID - 1
	// while internal buffer size > maximum size
	for int64(len(b.data)) > b.segment*b.maxCount {
		// CPU: suboptimal.
		b.firstID++
		b.data = b.data[b.segment:]
	}

	b.l.Unlock()
	return len(buf), nil
}

func (b *Buffer) acquireID(id int64) error {
	if len(b.data) == 0 {
		return ErrEmpty
	}
	if id < b.firstID || id > b.lastID {
		return ErrMiss
	}
	return nil
}

// ReadID reads semgent with provided id to w.
func (b *Buffer) ReadID(w io.Writer, id int64) (int, error) {
	b.l.Lock() // should be unlocked before w.Write call
	if err := b.acquireID(id); err != nil {
		b.l.Unlock()
		return 0, errors.Wrap(err, "bad id")
	}
	var buf []byte
	copy(buf, b.getSegment(id))
	b.l.Unlock()
	return w.Write(buf)
}

// Get writes segment with requested id into buf
func (b *Buffer) Get(buf []byte, id int64) error {
	if int64(len(buf)) < b.segment {
		return errors.Wrap(ErrBufferTooSmall, "bad buffer")
	}
	b.l.Lock()
	if err := b.acquireID(id); err != nil {
		return errors.Wrap(err, "bad id")
	}
	copy(buf[:b.segment], b.getSegment(id))
	b.l.Unlock()
	return nil
}

// LastID returns last segment id.
func (b *Buffer) LastID() int64 {
	b.l.Lock()
	defer b.l.Unlock()
	return b.lastID
}

// FirstID returns first segment id.
func (b *Buffer) FirstID() int64 {
	b.l.Lock()
	defer b.l.Unlock()
	return b.firstID
}
