package player

import (
	"io"
	"sync"

	log "github.com/Sirupsen/logrus"
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
)

// Buffer represents in-memory buffer for stream.
type Buffer struct {
	segment  int64
	maxCount int64
	count    int64
	lastID   int64
	firstID  int64
	l        sync.Mutex
	data     []byte
}

type Config struct {
	Segment int64
	Count   int64
	Start   int64
}

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
	}
}

func NewDefault() *Buffer {
	return New(Config{})
}

func (b *Buffer) SetCount(count int64) {
	b.l.Lock()
	b.maxCount = count
	b.l.Unlock()
}

func (b Buffer) SegmentSize() int64 {
	return b.segment
}

func (b Buffer) Count() int64 {
	return b.maxCount
}

func (b Buffer) Size() int {
	b.l.Lock()
	defer b.l.Unlock()
	return len(b.data)
}

func (b Buffer) getSegment(id int64) []byte {
	start := b.segment * (id - b.firstID)
	return b.data[start : b.segment+start]
}

// Write appends internal buffer with new data.r
func (b *Buffer) Write(buf []byte) (int, error) {
	if int64(len(buf)) > b.segment*b.maxCount {
		log.Warnf("write len(buf) %d > %d", len(buf), b.segment*b.maxCount)
		return 0, errors.Wrap(ErrTooLargeWrite, "failed to write")
	}
	b.l.Lock()
	b.data = append(b.data, buf...)
	b.count = int64(len(b.data)) / b.segment
	b.lastID = b.count + b.firstID
	for int64(len(b.data)) > b.segment*b.maxCount {
		b.firstID++
		b.data = b.data[b.segment:]
	}
	b.l.Unlock()
	return len(buf), nil
}

func (b *Buffer) acquireID(id int64) error {
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
		return ErrBufferTooSmall
	}
	b.l.Lock()
	if err := b.acquireID(id); err != nil {
		return errors.Wrap(err, "bad id")
	}
	copy(buf, b.getSegment(id))
	b.l.Unlock()
	return nil
}
