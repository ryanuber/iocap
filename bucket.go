package iocap

import "time"

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
		tokenCh: make(chan struct{}, opts.n),
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
		delay := b.drained.Add(b.opts.d).Sub(time.Now())
		time.Sleep(delay)
	}

	if time.Since(b.drained) >= b.opts.d {
		defer func() {
			b.drained = time.Now()
		}()
		for i := 0; i < b.opts.n; i++ {
			select {
			case <-b.tokenCh:
			default:
				return
			}
		}
	}
}
