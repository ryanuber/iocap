package iocap

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimitHandler(t *testing.T) {
	// Create some random data for the response body.
	data := make([]byte, 512)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Start the wrapped HTTP server with an applied rate limit.
	ts := httptest.NewServer(LimitHTTPHandler(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write(data)
		}), RateOpts{100 * time.Millisecond, 128}))
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
