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

// NewReader creates a new limited reader over the given source reader. The
// limit is the number of bytes allowed to be transferred per interval.
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

	b := make([]byte, 1)
	for n < cap(p) {
		bucket.wait()
		_, err = r.src.Read(b)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
		n += copy(p[n:], b)
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

// wait blocks until there is room in the bucket for a token insert.
func (b *bucket) wait() {
	b.tokenCh <- struct{}{}
}
