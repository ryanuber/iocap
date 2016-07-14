package iocap

import (
	"io"
	"time"
)

// Reader implements the io.Reader interface and limits the rate at which
// bytes come off of the underlying source reader.
type Reader struct {
	limit  int
	source io.Reader
}

// NewReader creates a new limited reader over the given source reader. The
// limit is the number of bytes allowed to be transferred per interval.
func NewReader(source io.Reader, limit int) *Reader {
	return &Reader{
		limit:  limit,
		source: source,
	}
}

// Read reads bytes off of the underlying source reader onto p with rate
// limiting. Reads until EOF or until p is filled.
func (r *Reader) Read(p []byte) (n int, err error) {
	b := make([]byte, 1)
	for i := 0; i <= cap(p); i++ {
		select {
		case <-time.After(time.Second / time.Duration(r.limit)):
			_, err = r.source.Read(b)
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				return
			}
			n += copy(p[n:], b)
		}
	}
	return
}
