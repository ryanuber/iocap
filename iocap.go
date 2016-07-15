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
		// Ask for enough space to fit all remaining bytes
		v := r.bucket.wait(len(p) - n)

		// Read from src into the byte range in p
		v, err = r.src.Read(p[n : n+v])

		// Count the actual number of bytes read.
		n += v

		// Return any errors from the underlying reader. Preserves the
		// underlying implementation's functionality.
		if err != nil {
			return
		}
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
		// Ask for enough space to write p completely.
		v := w.bucket.wait(len(p) - n)

		// Write from the byte offset on p into the writer.
		v, err = w.dst.Write(p[n : n+v])

		// Count the actual bytes written.
		n += v

		// Return any errors from the underlying writer. Preserves the
		// underlying implementation's functionality.
		if err != nil {
			return
		}
	}
	return
}

// RateOpts is used to encapsulate rate limiting options.
type RateOpts struct {
	// d is the time period of the rate
	d time.Duration

	// n is the number of bytes per interval
	n int
}

// PerSecond returns a RateOpts configured to allow n bytes per second.
func PerSecond(n int) RateOpts {
	return RateOpts{time.Second, n}
}
