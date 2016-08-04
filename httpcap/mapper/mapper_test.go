package mapper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandler(t *testing.T) {
	// Group by the request path
	g := func(r *http.Request) string {
		return r.URL.Path
	}

	// Return handlers that echo back the group name
	f := func(grp string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, grp)
		})
	}

	// Create the httpmap handler
	h := New(g, f, time.Second)

	// Requests are routed to their proper handlers.
	tcases := []string{"/foo", "/bar"}
	for _, tc := range tcases {
		req, err := http.NewRequest("GET", tc, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		resp := httptest.NewRecorder()
		h.ServeHTTP(resp, req)

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if v := string(body); !strings.Contains(v, tc) {
			t.Fatalf("expect body to contain %q\nactual: %q", tc, v)
		}
	}
}

func TestGroupByRequestIP(t *testing.T) {
	// Create the mock request.
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Non host:port RemoteAddr just gets returned raw.
	req.RemoteAddr = "foo"
	if v := GroupByRequestIP(req); v != "foo" {
		t.Fatalf("expect %q, actual %q", "foo", v)
	}

	// Conformant host:port returns just the IP.
	req.RemoteAddr = "127.0.0.1:1234"
	if v := GroupByRequestIP(req); v != "127.0.0.1" {
		t.Fatalf("expect %q, actual %q", "127.0.0.1", v)
	}

	// X-Forwarded-For returned first, only first IP in list.
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 2.3.4.5")
	if v := GroupByRequestIP(req); v != "1.2.3.4" {
		t.Fatalf("expect %q, actual %q", "1.2.3.4", v)
	}
}

func TestReap(t *testing.T) {
	// Group requests by path
	g := func(r *http.Request) string {
		return r.URL.Path
	}

	// Simple handler to return the timestamp at which the
	// handler instance was created.
	f := func(_ string) http.Handler {
		return stringHandler(time.Now().String())
	}

	// Create the http mapper.
	h := New(g, f, 100*time.Millisecond)

	// First request for group foo
	req1, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req1)

	body1, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Second request for group "bar". We will let this one
	// expire and check it later.
	req2, err := http.NewRequest("GET", "/bar", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req2)

	body2, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check that the handlers returned a different result
	if string(body1) == string(body2) {
		t.Fatal("expect responses to be different")
	}

	// Wait for a bit, but don't let the foo handler expire.
	time.Sleep(50 * time.Millisecond)

	// Request for foo again
	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req1)

	body3, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if string(body1) != string(body3) {
		t.Fatal("should not reap foo")
	}

	// Wait a while longer so that bar expires
	time.Sleep(60 * time.Millisecond)

	// Request bar again
	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req2)

	body4, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if string(body2) == string(body4) {
		t.Fatal("should reap bar")
	}
}

type stringHandler string

func (h stringHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, string(h))
}
