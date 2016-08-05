package iocap

import (
	"io"
	"time"
)

const (
	_  = (1 << (10 * iota)) / 8
	Kb // Kilobit
	Mb // Megabit
	Gb // Gigabit
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
		v := r.bucket.insert(len(p) - n)

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

// SetRate is used to dynamically set the rate options on the reader.
func (r *Reader) SetRate(opts RateOpts) {
	r.bucket.setRate(opts)
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
		v := w.bucket.insert(len(p) - n)

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

// SetRate is used to dynamically set the rate options on the writer.
func (w *Writer) SetRate(opts RateOpts) {
	w.bucket.setRate(opts)
}

// RateOpts is used to encapsulate rate limiting options.
type RateOpts struct {
	// Interval is the time period of the rate
	Interval time.Duration

	// Size is the number of bytes per interval
	Size int
}

// perSecond is an internal helper to calculate rates.
func perSecond(n, base float64) RateOpts {
	return RateOpts{
		Interval: time.Second,
		Size:     int(n * base),
	}
}

// Kbps returns a RateOpts configured for n kilobits per second.
func Kbps(n float64) RateOpts {
	return perSecond(n, Kb)
}

// Mbps returns a RateOpts configured for n megabits per second.
func Mbps(n float64) RateOpts {
	return perSecond(n, Mb)
}

// Gbps returns a RateOpts configured for n gigabits per second.
func Gbps(n float64) RateOpts {
	return perSecond(n, Gb)
}

// Group is used to group multiple readers and/or writers onto the same bucket,
// thus enforcing the rate limit across multiple independent processes.
type Group struct {
	bucket *bucket
}

// NewGroup creates a new rate limiting group with the specific rate.
func NewGroup(opts RateOpts) *Group {
	return &Group{newBucket(opts)}
}

// SetRate is used to dynamically update the rate options of the group.
func (g *Group) SetRate(opts RateOpts) {
	g.bucket.setRate(opts)
}

// NewWriter creates and returns a new writer in the group.
func (g *Group) NewWriter(dst io.Writer) *Writer {
	return &Writer{
		dst:    dst,
		bucket: g.bucket,
	}
}

// NewReader creates and returns a new reader in the group.
func (g *Group) NewReader(src io.Reader) *Reader {
	return &Reader{
		src:    src,
		bucket: g.bucket,
	}
}
