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
	opts RateOpts
	src  io.Reader
}

// NewReader wraps src in a new rate limited reader.
func NewReader(src io.Reader, opts RateOpts) *Reader {
	return &Reader{
		opts: opts,
		src:  src,
	}
}

// Read reads bytes off of the underlying source reader onto p with rate
// limiting. Reads until EOF or until p is filled.
func (r *Reader) Read(p []byte) (n int, err error) {
	bucket := newBucket(r.opts)
	defer bucket.stop()

	for n < len(p) {
		v := bucket.wait(len(p) - n)
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
	opts RateOpts
	dst  io.Writer
}

// NewWriter wraps dst in a new rate limited writer.
func NewWriter(dst io.Writer, opts RateOpts) *Writer {
	return &Writer{
		opts: opts,
		dst:  dst,
	}
}

// Write writes len(p) bytes onto the underlying io.Writer, respecting the
// configured rate limit options.
func (w *Writer) Write(p []byte) (n int, err error) {
	bucket := newBucket(w.opts)
	defer bucket.stop()

	for n < len(p) {
		v := bucket.wait(len(p) - n)
		v, err = w.dst.Write(p[n : n+v])
		if err != nil {
			return
		}
		n += v
	}
	return
}

// PerMinute returns a RateOpts configured for the given rate per minute.
func PerMinute(n int) RateOpts {
	return RateOpts{time.Minute, n}
}

// PerSecond returns a RateOpts configured for the given rate per second.
func PerSecond(n int) RateOpts {
	return RateOpts{time.Second, n}
}

// bucket is used to guard io reads and writes using a simple timer.
type bucket struct {
	tokenCh chan struct{}
	doneCh  chan struct{}
}

// newBucket creates a new token bucket with the specified rate. The
// rate is the number of bytes per second
func newBucket(opts RateOpts) *bucket {
	b := &bucket{
		tokenCh: make(chan struct{}, opts.N),
		doneCh:  make(chan struct{}),
	}
	go func() {
		for {
			select {
			case <-b.doneCh:
				return
			case <-time.After(opts.D / time.Duration(opts.N)):
				select {
				case <-b.tokenCh:
				case <-b.doneCh:
					return
				}
			}
		}
	}()
	return b
}

// stop stops the goroutine which drains the bucket.
func (b *bucket) stop() {
	close(b.doneCh)
}

// wait is used to wait for n tokens to fit into the bucket. The token
// insert is best-effort, and the actual number of tokens inserted is
// returned. This allows grabbing a bulk of tokens in a single pass.
// wait will block until at least one token is inserted.
func (b *bucket) wait(n int) int {
	v := 0
	for i := 0; i < n; i++ {
		select {
		case b.tokenCh <- struct{}{}:
			v++
		default:
			if i > 0 {
				break
			}
			b.tokenCh <- struct{}{}
			v++
		}
	}
	return v
}
