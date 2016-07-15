package iocap

import (
	"reflect"
	"testing"
	"time"
)

func TestNewBucket(t *testing.T) {
	opts := RateOpts{0, 256}
	b := newBucket(opts)
	if n := cap(b.tokenCh); n != opts.Size {
		t.Fatalf("expect size 256, got %d", n)
	}
	if !reflect.DeepEqual(b.opts, opts) {
		t.Fatalf("expect: %#v\nactual: %#v", opts, b.opts)
	}
}

func TestBucketWait(t *testing.T) {
	// First create a bucket and exhaust the tokenCh
	b := newBucket(RateOpts{Interval: 100 * time.Millisecond, Size: 256})
	for i := 0; i < 256; i++ {
		b.tokenCh <- struct{}{}
	}

	// Returns immediately if tokens are all inserted
	start := time.Now()
	n := b.wait(256)
	if time.Since(start) > 10*time.Millisecond {
		t.Fatal("should insert immediately")
	}
	if n != 256 {
		t.Fatalf("expect 256, got: %d", n)
	}

	// Next token insert should block until the drain interval
	n = b.wait(128)
	if time.Since(start) < 100*time.Millisecond {
		t.Fatal("should block")
	}
	if n != 128 {
		t.Fatalf("expect 128, got: %d", n)
	}

	// Inserting tokens to a non-empty bucket returns fast
	// once we start to overflow.
	start = time.Now()
	n = b.wait(256)
	if time.Since(start) > 10*time.Millisecond {
		t.Fatal("should insert immediately")
	}
	if n != 128 {
		t.Fatalf("expect 128, got: %d", n)
	}
}

func TestBucketDrain(t *testing.T) {
	b := newBucket(RateOpts{Interval: 100 * time.Millisecond, Size: 256})

	// Place a token in the bucket for draining
	b.wait(1)

	// Doesn't drain if the expiration isn't passed.
	b.drain(false)
	if len(b.tokenCh) != 1 {
		t.Fatal("should not drain tokens")
	}

	// Waits for the next interval and drains when wait is true
	start := time.Now()
	b.drain(true)
	if time.Since(start) < 100*time.Millisecond {
		t.Fatal("should block")
	}
	if len(b.tokenCh) != 0 {
		t.Fatal("should drain tokens")
	}
}
