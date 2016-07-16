package iocap

import (
	"sync"
	"time"
)

// bucket is a simple "leaky bucket" abstraction to provide a way to
// limit the number of operations (in this case, byte reads/writes)
// allowed within a given interval.
type bucket struct {
	tokenCh chan struct{}
	opts    RateOpts
	drained time.Time

	sync.RWMutex
}

// newBucket creates a new bucket to use for readers and writers.
func newBucket(opts RateOpts) *bucket {
	return &bucket{
		tokenCh: make(chan struct{}, opts.Size),
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
	b.RLock()
	last := b.drained
	b.RUnlock()

	switch {
	case time.Since(last) >= b.opts.Interval:
		b.Lock()
		defer b.Unlock()

		// Make sure the timestamp was not updated; prevents a time-of-
		// check vs. time-of-use error.
		if !b.drained.Equal(last) {
			return
		}

		// Update the drain timestamp at exit.
		defer func() {
			b.drained = time.Now()
		}()

		// Drain the bucket.
		for i := 0; i < b.opts.Size; i++ {
			select {
			case <-b.tokenCh:
			default:
				return
			}
		}

	case wait:
		delay := last.Add(b.opts.Interval).Sub(time.Now())
		time.Sleep(delay)
		b.drain(false)
	}
}
