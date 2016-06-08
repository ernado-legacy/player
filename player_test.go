package player

import (
	"testing"
	"io"
	"math/rand"
	"bytes"
)

var (
	source = rand.NewSource(1488)
)

func Rand() io.Reader {
	return rand.New(source)
}

func TestBuffer_Write(t *testing.T) {
	r := Rand()
	b := NewDefault()
	if _, err := io.CopyN(b, r, b.SegmentSize() * 4); err != nil {
		t.Error(err)
	}
}

func TestBuffer_Get(t *testing.T) {
	r := Rand()
	b := NewDefault()
	if _, err := io.CopyN(b, r, b.SegmentSize() * 4); err != nil {
		t.Error(err)
	}
	buf := new(bytes.Buffer)
	if _, err := b.ReadID(buf, 3); err != nil {
		t.Error(err)
	}
}
