package iocap

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	// Create some random data for our reader.
	data := make([]byte, 512)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("err: %v", err)
	}
	buf := bytes.NewBuffer(data)

	// Create the Reader with a rate limit applied.
	r := NewReader(buf, RateOpts{Interval: 100 * time.Millisecond, Size: 128})
	out := make([]byte, 512)

	// Record the start time and execute the read.
	start := time.Now()
	n, err := r.Read(out)

	// Check that we actually rate limited the read. 300ms because
	// initially we can read 128 bytes, then 3 bucket drains block
	// at 100ms a pop.
	if d := time.Since(start); d < 300*time.Millisecond {
		t.Fatalf("read returned to quickly in %s", d)
	}

	// Check the return values and data.
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 512 {
		t.Fatalf("expect 512, got: %d", n)
	}
	if !bytes.Equal(data, out) {
		t.Fatal("unexpected data read")
	}
}

func TestReaderSetRate(t *testing.T) {
	// Create a new reader with unlimited rate.
	r := NewReader(new(bytes.Buffer), Unlimited)

	// Set the rate to something and check it.
	expect := RateOpts{time.Second, 1}
	r.SetRate(expect)
	if v := r.bucket.opts; v != expect {
		t.Fatalf("expect %v\nactual: %v", expect, v)
	}
}

func TestWriter(t *testing.T) {
	// Create some random data to write.
	data := make([]byte, 512)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create the writer with an applied rate limit.
	buf := new(bytes.Buffer)
	w := NewWriter(buf, RateOpts{Interval: 100 * time.Millisecond, Size: 128})

	// Record the start time and perform the write.
	start := time.Now()
	n, err := w.Write(data)

	// Check that we rate limited the write.
	if d := time.Since(start); d < 300*time.Millisecond {
		t.Fatalf("write returned too quickly in %s", d)
	}

	// Check errors and data values.
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 512 {
		t.Fatalf("expect 512, got: %d", n)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Fatal("unexpected data written")
	}
}

func TestWriterSetRate(t *testing.T) {
	// Create a new writer with unlimited rate.
	w := NewWriter(new(bytes.Buffer), Unlimited)

	// Set the rate to something and check it.
	expect := RateOpts{time.Second, 1}
	w.SetRate(expect)
	if v := w.bucket.opts; v != expect {
		t.Fatalf("expect %v\nactual: %v", expect, v)
	}
}

func TestGroup(t *testing.T) {
	// Create the rate limiting group.
	g := NewGroup(RateOpts{Interval: 100 * time.Millisecond, Size: 8})

	// Create two buffers; one for the reader, one for the writer. The
	// idea is that these are completely separate buffers, maybe completely
	// unrelated in terms of the operation being performed on them.
	// Regardless, we want to ensure that operations across both share the
	// rate limit of the group.
	bufW := new(bytes.Buffer)
	bufR := new(bytes.Buffer)

	// Set up the data for the reader/writer
	in := []byte("hello world!")
	bufR.Write(in)
	out := make([]byte, len(in))

	// Create a reader and writer in the group against the buffers.
	w := g.NewWriter(bufW)
	r := g.NewReader(bufR)

	// Mark the start time and perform the read and write in parallel.
	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		if _, err := w.Write(in); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	go func() {
		defer wg.Done()

		if _, err := r.Read(out); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	// Wait for both operations to finish.
	wg.Wait()

	// Make sure we blocked for at least 2 intervals. If each operation
	// had its own 8B/s limit, we would finish in ~100ms, but because we
	// are sharing the limit, pushing 24 bytes through is going to require
	// two bucket drains before it completes.
	if d := time.Since(start); d < 200*time.Millisecond {
		t.Fatalf("finished too quickly in %s", d)
	}
}

func TestGroupSetRate(t *testing.T) {
	// Create a new group with unlimited rate.
	g := NewGroup(Unlimited)

	// Set the rate to something and check it.
	expect := RateOpts{1, 1}
	g.SetRate(expect)
	if v := g.bucket.opts; v != expect {
		t.Fatalf("expect: %v\nactual: %v", expect, v)
	}
}

func TestKbps(t *testing.T) {
	ro := Kbps(128)
	if ro.Interval != time.Second {
		t.Fatalf("expect 1s, got: %s", ro.Interval)
	}
	if expect := Kb * 128; expect != ro.Size {
		t.Fatalf("expect %d, got: %d", expect, ro.Size)
	}
}

func TestMbps(t *testing.T) {
	ro := Mbps(128)
	if ro.Interval != time.Second {
		t.Fatalf("expect 1s, got: %s", ro.Interval)
	}
	if expect := Mb * 128; expect != ro.Size {
		t.Fatalf("expect %d, got: %d", expect, ro.Size)
	}
}

func TestGbps(t *testing.T) {
	ro := Gbps(128)
	if ro.Interval != time.Second {
		t.Fatalf("expect 1s, got: %s", ro.Interval)
	}
	if expect := Gb * 128; expect != ro.Size {
		t.Fatalf("expect %d, got: %d", expect, ro.Size)
	}
}

func ExampleReader() {
	// Create a buffer to read from.
	buf := bytes.NewBufferString("hello world!")

	// Create the rate limited reader.
	rate := Kbps(512)
	r := NewReader(buf, rate)

	// Read from the reader.
	out := make([]byte, buf.Len())
	n, err := r.Read(out)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(n, string(out))
	// Output: 12 hello world!
}

func ExampleWriter() {
	// Create the buffer to write to.
	buf := new(bytes.Buffer)

	// Create the rate limited writer.
	rate := Kbps(512)
	r := NewWriter(buf, rate)

	// Write data into the writer.
	n, err := r.Write([]byte("hello world!"))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(n, buf.String())
	// Output: 12 hello world!
}

func ExampleGroup() {
	// Create a rate limiting group.
	rate := Kbps(512)
	g := NewGroup(rate)

	// Create a new reader and writer on the group.
	buf := new(bytes.Buffer)
	r := g.NewReader(buf)
	w := g.NewWriter(buf)

	// Reader and writer are rate limited together
	n, err := w.Write([]byte("hello world!"))
	if err != nil {
		fmt.Println(err)
		return
	}
	out := make([]byte, n)
	_, err = r.Read(out)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(out))
	// Output: hello world!
}
