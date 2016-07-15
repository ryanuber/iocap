package iocap

import (
	"io"
	"time"
)

// Reader implements the io.Reader interface and limits the rate at which
// bytes come off of the underlying source reader.
type Reader struct {
	src    io.Reader
	bucket *bucket
}

// NewReader wraps src in a new rate limited reader.
func NewReader(src io.Reader, opts RateOpts) *Reader {
	return &Reader{
		src:    src,
		bucket: newBucket(opts),
	}
}

// Read reads bytes off of the underlying source reader onto p with rate
// limiting. Reads until EOF or until p is filled.
func (r *Reader) Read(p []byte) (n int, err error) {
	for n < len(p) {
		v := r.bucket.wait(len(p) - n)
		v, err = r.src.Read(p[n : n+v])
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
		n += v
	}
	return
}

// Writer implements the io.Writer interface and limits the rate at which
// bytes are written to the underlying writer.
type Writer struct {
	dst    io.Writer
	bucket *bucket
}

// NewWriter wraps dst in a new rate limited writer.
func NewWriter(dst io.Writer, opts RateOpts) *Writer {
	return &Writer{
		dst:    dst,
		bucket: newBucket(opts),
	}
}

// Write writes len(p) bytes onto the underlying io.Writer, respecting the
// configured rate limit options.
func (w *Writer) Write(p []byte) (n int, err error) {
	for n < len(p) {
		v := w.bucket.wait(len(p) - n)
		v, err = w.dst.Write(p[n : n+v])
		if err != nil {
			return
		}
		n += v
	}
	return
}

// RateOpts is used to encapsulate rate limiting options.
type RateOpts struct {
	D time.Duration
	N int
}

// PerSecond returns a RateOpts configured to allow n bytes per second.
func PerSecond(n int) RateOpts {
	return RateOpts{time.Second, n}
}
