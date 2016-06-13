package player

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
	"math/rand"
	"testing"
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
	if _, err := io.CopyN(b, r, b.SegmentSize()*4); err != nil {
		t.Error(err)
	}
	buf := make([]byte, b.SegmentSize()*b.Count()+b.SegmentSize())
	if _, err := b.Write(buf); errors.Cause(err) != ErrTooLargeWrite {
		t.Error(err, "should be", ErrTooLargeWrite)
	}
}

func TestBuffer_SetCount(t *testing.T) {
	b := NewDefault()
	c := b.Count() + 5
	buf := make([]byte, c*b.SegmentSize())
	if _, err := b.Write(buf); errors.Cause(err) != ErrTooLargeWrite {
		t.Error(err, "should be", ErrTooLargeWrite)
	}
	b.SetCount(c)
	if _, err := b.Write(buf); err != nil {
		t.Error(err)
	}
}

func TestBuffer_Size(t *testing.T) {
	b := New(Config{
		Count:   12,
		Segment: 512,
	})
	buf := make([]byte, 10*512)
	if _, err := b.Write(buf); err != nil {
		t.Error(err)
	}
	if b.Size() != 512*10 {
		t.Error("bad size")
	}
}

func TestBuffer_Collection(t *testing.T) {
	b := New(Config{
		Count:   12,
		Segment: 512,
	})
	buf := make([]byte, 12*512)
	if _, err := b.Write(buf); err != nil {
		t.Error(err)
	}
	buf = buf[:2*512]
	if _, err := b.Write(buf); err != nil {
		t.Error(err)
	}
}

func TestBuffer_ReadID(t *testing.T) {
	r := Rand()
	b := NewDefault()
	if _, err := io.CopyN(b, r, b.SegmentSize()*4); err != nil {
		t.Error(err)
	}
	buf := new(bytes.Buffer)
	if _, err := b.ReadID(buf, 3); err != nil {
		t.Error(err)
	}
}

func TestBuffer_Get(t *testing.T) {
	r := Rand()
	b := NewDefault()
	if _, err := io.CopyN(b, r, b.SegmentSize()*4); err != nil {
		t.Error(err)
	}
	buf := make([]byte, b.SegmentSize())
	if err := b.Get(buf, 3); err != nil {
		t.Error(err)
	}

	buf = buf[:b.SegmentSize()/2]
	if err := b.Get(buf, 3); errors.Cause(err) != ErrBufferTooSmall {
		t.Error(err, "should be", ErrBufferTooSmall)
	}
	buf = buf[:b.SegmentSize()]
	if err := b.Get(buf, b.Count()*2); errors.Cause(err) != ErrMiss {
		t.Error(err, "should be", ErrMiss)
	}
}

func TestBuffer_ID(t *testing.T) {
	b := NewDefault()
	buf := make([]byte, b.SegmentSize()*3)
	// 0, 1, 2
	if _, err := b.Write(buf); err != nil {
		t.Error(err)
	}
	if b.LastID() != 2 {
		t.Error("bad last id", b.LastID())
	}
	if err := b.Get(buf, b.LastID()); err != nil {
		t.Error(err)
	}
	if b.FirstID() != 0 {
		t.Error("bad first id", b.FirstID())
	}
}

func TestBuffer_ErrMiss(t *testing.T) {
	b := NewDefault()
	buf := new(bytes.Buffer)
	if _, err := b.ReadID(buf, 1); errors.Cause(err) != ErrEmpty {
		t.Error(err, "should be", ErrEmpty)
	}
}

func TestError(t *testing.T) {
	if Error("error").Error() != "error" {
		t.Error("bad Error")
	}
	if Error("error").String() != "err: error" {
		t.Error("bad String for Error")
	}
}