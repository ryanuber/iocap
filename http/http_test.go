package httpcap

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ryanuber/iocap"
)

func TestLimitHandler(t *testing.T) {
	// Create some random data for the response body.
	data := make([]byte, 512)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Start the wrapped HTTP server with an applied rate limit.
	ts := httptest.NewServer(LimitHandler(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write(data)
		}), iocap.RateOpts{Interval: 100 * time.Millisecond, Size: 128}))
	defer ts.Close()

	// Record the start time and perform the request.
	start := time.Now()
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer resp.Body.Close()

	// Check that the response was delayed. 300ms because 128 bytes
	// can be read immediately, then 3 more intervals of 128 bytes
	// with a 100ms delay between.
	if d := time.Since(start); d < 300*time.Millisecond {
		t.Fatalf("response returned too quickly in %s", d)
	}

	// Check the response body.
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !bytes.Equal(out, data) {
		t.Fatal("unexpected data returned")
	}
}

func ExampleLimitHandler() {
	// Create a normal HTTP handler to serve data.
	h := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world!"))
	}))

	// Wrap the handler with a rate limit.
	rate := iocap.PerSecond(1024 * 128) // 128K/s
	h = LimitHandler(h, rate)

	// Start a test server using the rate limited handler.
	ts := httptest.NewServer(h)
	defer ts.Close()

	// Make a request to the server.
	resp, err := http.Get(ts.URL)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(body))
	// Output: hello world!
}

func ExampleLimitResponseWriter() {
	// Create an HTTP handler with a rate limited response writer.
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = LimitResponseWriter(w, iocap.PerSecond(1024*128)) // 128K/s
		w.Write([]byte("hello world!"))
	})

	// Start a test server using the handler.
	ts := httptest.NewServer(h)
	defer ts.Close()

	// Make a request to the server.
	resp, err := http.Get(ts.URL)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(body))
	// Output: hello world!
}
