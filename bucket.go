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

	// Tokens is the number of tokens present in the bucket. A simple int is
	// used to allow for faster token acquisition, rather than a channel.
	// Arguably, due to the blocking nature of iocap, a channel may be
	// theoretically more appropriate for this use. The reality pitfall is
	// that billions of channel reads are far more expensive than taking a
	// lock and doing basic math.
	tokens int

	l sync.RWMutex
}

// newBucket creates a new bucket to use for readers and writers.
func newBucket(opts RateOpts) *bucket {
	return &bucket{
		opts: opts,
	}
}

// insert performs a best-effort token insert of n tokens. v contains
// the number of tokens inserted, which will differ from n if the
// bucket overflows. insert will block until at least one token is
// successfully inserted.
func (b *bucket) insert(n int) (v int) {
	// Call a non-blocking drain up-front to make room for tokens.
	b.drain(false)

INSERT:
	var remain int

	b.l.RLock()
	tokens := b.tokens
	opts := b.opts
	b.l.RUnlock()

	switch {
	case opts == Unlimited:
		// No limit should be applied.
		return n

	case tokens == opts.Size:
		// Bucket is full. Call a blocking drain to wait for the next
		// drain interval (earliest we can insert more tokens).
		b.drain(true)
		goto INSERT

	case tokens+n > opts.Size:
		// Some tokens, but not all, were inserted. The bucket is now
		// full and subsequent inserts will overflow and block.
		v = opts.Size - tokens
		remain = opts.Size

	default:
		// All tokens inserted successfully.
		v = n
		remain = tokens + n
	}

	b.l.Lock()

	// Check if the token count was modified before the lock
	// was acquired.
	if b.tokens != tokens {
		b.l.Unlock()
		goto INSERT
	}

	b.tokens = remain
	b.l.Unlock()
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
	b.l.RLock()
	last := b.drained
	interval := b.opts.Interval
	b.l.RUnlock()

	switch {
	case time.Since(last) >= interval:
		b.l.Lock()
		defer b.l.Unlock()

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
		delay := last.Add(interval).Sub(time.Now())
		time.Sleep(delay)
		b.drain(false)
	}
}

// setRate safely replaces the RateOpts on the bucket.
func (b *bucket) setRate(opts RateOpts) {
	b.l.Lock()
	b.opts = opts
	b.l.Unlock()
}
