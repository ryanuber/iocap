package iocap

import (
	"io"
	"time"
)

// RateOpts is used to encapsulate rate limiting options.
type RateOpts struct {
	D time.Duration
	N int
}

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

// PerSecond returns a RateOpts configured to allow n bytes per second.
func PerSecond(n int) RateOpts {
	return RateOpts{time.Second, n}
}

// bucket is a simple "leaky bucket" abstraction to provide a way to
// limit the number of operations (in this case, byte reads/writes)
// allowed within a given interval.
type bucket struct {
	tokenCh chan struct{}
	opts    RateOpts
	drained time.Time
}

// newBucket creates a new bucket to use for readers and writers.
func newBucket(opts RateOpts) *bucket {
	return &bucket{
		tokenCh: make(chan struct{}, opts.N),
		opts:    opts,
	}
}

// wait is used to wait for n tokens to fit into the bucket. The token
// insert is best-effort, and the actual number of tokens inserted is
// returned. This allows grabbing a bulk of tokens in a single pass.
// wait will block until at least one token is inserted.
func (b *bucket) wait(n int) (v int) {
	// Call a non-blocking drain up-front to make room for tokens.
	b.drain(false)

	for v < n {
		select {
		case b.tokenCh <- struct{}{}:
			v++
		default:
			if v == 0 {
				// Call a blocking drain, because the bucket cannot
				// make progress until the drain interval arrives.
				b.drain(true)
				continue
			}
			return
		}
	}
	return
}

// drain is used to drain the bucket of tokens. If wait is true, drain
// will wait until the next drain cycle and then continue. Otherwise,
// drain only drains the bucket if it is due.
func (b *bucket) drain(wait bool) {
	if wait {
		delay := b.drained.Add(b.opts.D).Sub(time.Now())
		time.Sleep(delay)
	}

	if time.Since(b.drained) >= b.opts.D {
		defer func() {
			b.drained = time.Now()
		}()
		for i := 0; i < b.opts.N; i++ {
			select {
			case <-b.tokenCh:
			default:
				return
			}
		}
	}
}
