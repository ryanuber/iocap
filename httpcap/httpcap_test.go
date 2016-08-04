package httpcap

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ryanuber/iocap"
)

func TestHandler(t *testing.T) {
	// Create some random data for the response body.
	data := make([]byte, 512)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create a normal HTTP handler to return data.
	h := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))

	// Wrap the handler with a rate limit.
	rate := iocap.RateOpts{Interval: 100 * time.Millisecond, Size: 128}
	h = Handler(h, rate)

	// Start the wrapped HTTP server.
	ts := httptest.NewServer(h)
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

func TestGroupHandler(t *testing.T) {
	// Create some random data for the response body.
	data := make([]byte, 512)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create a normal HTTP handler to return data.
	h := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))

	// Wrap the handler with a rate limit.
	rate := iocap.RateOpts{Interval: 100 * time.Millisecond, Size: 128}
	group := iocap.NewGroup(rate)
	h = GroupHandler(h, group)

	// Start the wrapped HTTP server.
	ts := httptest.NewServer(h)
	defer ts.Close()

	// Record the start time and perform two requests. They should share
	// the same rate limit quota.
	start := time.Now()

	// Perform two requests in parallel. The shared rate limit should
	// force this to take longer than if the requests were rate
	// limited in isolation of each other.
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()

			resp, err := http.Get(ts.URL)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			defer resp.Body.Close()

			// Check the response body.
			out, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("err: %v", err)
			}

			if !bytes.Equal(out, data) {
				t.Fatal("unexpected data returned")
			}
		}()
	}

	// Wait for both requests to finish
	wg.Wait()

	// Check the duration of the request. Should be higher than 600ms if
	// the rate was applied to both requests.
	if d := time.Since(start); d < 600*time.Millisecond {
		t.Fatalf("response returned too quickly in %s", d)
	}
}

func ExampleHandler() {
	// Create a normal HTTP handler to serve data.
	h := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world!"))
	}))

	// Wrap the handler with a rate limit.
	rate := iocap.PerSecond(1024 * 128) // 128K/s
	h = Handler(h, rate)

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

func ExampleGroupHandler() {
	// Create a normal HTTP handler to serve data.
	h := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world!"))
	}))

	// Wrap the handler with a rate limit group. All requests to this handler
	// will share the rate below.
	rate := iocap.PerSecond(1024 * 128) // 128K/s
	group := iocap.NewGroup(rate)
	h = GroupHandler(h, group)

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
