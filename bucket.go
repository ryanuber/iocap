package iocap

import (
	"sync"
	"time"
)

// bucket is a simple "leaky bucket" abstraction to provide a way to
// limit the number of operations (in this case, byte reads/writes)
// allowed within a given interval.
type bucket struct {
	opts    RateOpts
	drained time.Time

	// Tokens is the number of tokens present in the bucket. This number
	// is guarded by the tokenLock mutex. A simple int is used to allow
	// for faster token acquisition, rather than a channel. Arguably, due
	// to the blocking nature of iocap, a channel may be theoretically
	// more appropriate for this use. The reality pitfall is that billions
	// of channel reads are far more expensive than taking a lock and
	// doing basic math.
	tokens    int
	tokenLock sync.RWMutex
}

// newBucket creates a new bucket to use for readers and writers.
func newBucket(opts RateOpts) *bucket {
	return &bucket{
		opts: opts,
	}
}

// wait is used to wait for n tokens to fit into the bucket. The token
// insert is best-effort, and the actual number of tokens inserted is
// returned. This allows grabbing a bulk of tokens in a single pass.
// wait will block until at least one token is inserted.
func (b *bucket) wait(n int) (v int) {
	// Call a non-blocking drain up-front to make room for tokens.
	b.drain(false)

	for v == 0 {
		v = b.insert(n)
		if v == 0 {
			// Call a blocking drain, because the bucket cannot
			// make progress until the drain interval arrives.
			b.drain(true)
		}
	}
	return
}

// insert attempts to insert n tokens into the bucket, returning the
// actual number of tokens inserted. If the bucket overflows, v can
// differ from n.
func (b *bucket) insert(n int) (v int) {
INSERT:
	var remain int

	b.tokenLock.RLock()
	tokens := b.tokens
	b.tokenLock.RUnlock()

	switch {
	case tokens == b.opts.Size:
		v = 0
		return

	case tokens+n > b.opts.Size:
		v = b.opts.Size - tokens
		remain = b.opts.Size

	default:
		v = n
		remain = tokens + n
	}

	b.tokenLock.Lock()

	// Check if the token count was modified before the lock
	// was acquired.
	if b.tokens != tokens {
		b.tokenLock.Unlock()
		goto INSERT
	}

	b.tokens = remain
	b.tokenLock.Unlock()
	return
}

// drain is used to drain the bucket of tokens. If wait is true, drain
// will wait until the next drain cycle and then continue. Otherwise,
// drain only drains the bucket if it is due.
//
// This implementation is heavy-handed in that it brackets "leaking" tokens
// to the full duration of the configured interval. In other words, the
// bucket leaks not in single drops, but rather multiples, and only when the
// token drain window has elapsed. This side-steps near-hot-looping with
// dense token expiration (short interval + high size) and heavy lock
// contention. A possible enhancement would be to make this more granular.
func (b *bucket) drain(wait bool) {
	b.tokenLock.RLock()
	last := b.drained
	b.tokenLock.RUnlock()

	switch {
	case time.Since(last) >= b.opts.Interval:
		b.tokenLock.Lock()
		defer b.tokenLock.Unlock()

		// Make sure the timestamp was not updated; prevents a time-of-
		// check vs. time-of-use error.
		if !b.drained.Equal(last) {
			return
		}

		// Drain the bucket.
		b.tokens = 0

		// Update the drain timestamp.
		b.drained = time.Now()

	case wait:
		delay := last.Add(b.opts.Interval).Sub(time.Now())
		time.Sleep(delay)
		b.drain(false)
	}
}
