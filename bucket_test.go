package iocap

import (
	"testing"
	"time"
)

func TestBucketInsert(t *testing.T) {
	// First create a bucket.
	b := newBucket(RateOpts{Interval: 100 * time.Millisecond, Size: 256})

	// Returns immediately if tokens are all inserted
	start := time.Now()
	n := b.insert(256)
	if time.Since(start) > 10*time.Millisecond {
		t.Fatal("should insert immediately")
	}
	if n != 256 {
		t.Fatalf("expect 256, got: %d", n)
	}

	// Next token insert should block until the drain interval
	n = b.insert(128)
	if time.Since(start) < 100*time.Millisecond {
		t.Fatal("should block")
	}
	if n != 128 {
		t.Fatalf("expect 128, got: %d", n)
	}

	// Inserting tokens to a non-empty bucket returns fast
	// once we start to overflow.
	start = time.Now()
	n = b.insert(256)
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
	b.insert(1)

	// Doesn't drain if the expiration isn't passed.
	b.drain(false)
	if b.tokens != 1 {
		t.Fatal("should not drain tokens")
	}

	// Waits for the next interval and drains when wait is true
	start := time.Now()
	b.drain(true)
	if time.Since(start) < 100*time.Millisecond {
		t.Fatal("should block")
	}
	if b.tokens != 0 {
		t.Fatal("should drain tokens")
	}
}

func TestBucketSetRate(t *testing.T) {
	r1 := RateOpts{Interval: 100 * time.Millisecond, Size: 256}
	r2 := RateOpts{Interval: 500 * time.Millisecond, Size: 512}

	b := newBucket(r1)

	// Check that we can insert exactly r1's capacity
	if v := b.insert(1024); v != 256 {
		t.Fatalf("expect 256, got: %d", v)
	}

	// Update the rate opts in-place
	b.setRate(r2)

	// Check that we can insert more tokens. Make sure the previous tokens
	// are still intact in the bucket.
	if v := b.insert(1024); v != 256 {
		t.Fatalf("expect 256, got: %d", v)
	}
}
