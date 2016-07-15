package iocap

import (
	"bytes"
	"crypto/rand"
	"fmt"
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

func TestPerSecond(t *testing.T) {
	ro := PerSecond(128)
	if ro.Interval != time.Second {
		t.Fatalf("expect 1s, got: %s", ro.Interval)
	}
	if ro.Size != 128 {
		t.Fatalf("expect 128, got: %d", ro.Size)
	}
}

func ExampleReader() {
	// Create a buffer to read from.
	buf := bytes.NewBufferString("hello world!")

	// Create the rate limited reader.
	rate := PerSecond(8) // 8B/s
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
	rate := PerSecond(8) // 8B/s
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
