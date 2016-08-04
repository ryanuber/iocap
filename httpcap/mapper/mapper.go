package mapper

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// handler is a proxy http.Handler implementation, which allows splitting
// incoming requests off to different handlers based on the parameters of
// the request.
type handler struct {
	grouper RequestGrouper
	factory HandlerFactory

	// Group handlers and associated reap timers.
	groups    map[string]http.Handler
	groupReap map[string]*time.Timer
	reapDelay time.Duration

	l sync.Mutex
}

// New creates a new grouping HTTP handler. Requests are grouped by g,
// each group's http handler is created by f, and the group handler instance
// expires after duration r. After expiration, subsequent requests will create
// a replacement handler using the factory. The expiration timer for a group
// is reset at the beginning of each request. A zero value for r means no
// expiration, and is only recommended when grouping on commonly-seen request
// parameter values (request path, headers, etc). A good rule of thumb is to
// set the reap time to 2x the estimated max request duration.
func New(g RequestGrouper, f HandlerFactory, r time.Duration) http.Handler {
	return &handler{
		grouper:   g,
		factory:   f,
		groups:    make(map[string]http.Handler),
		groupReap: make(map[string]*time.Timer),
		reapDelay: r,
	}
}

// ServeHTTP implements the http.Handler interface using request's
// matching grouped http.Handler.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// First get the group key
	group := h.grouper(r)

	// Get the group handler
	hand := h.handler(group)

	// Service the request
	hand.ServeHTTP(w, r)
}

// handler looks up or creates a new http.Handler for the given group. If
// there is a reap timer configured, the timer is either started or reset.
func (h *handler) handler(group string) http.Handler {
	h.l.Lock()
	defer h.l.Unlock()

	hand, ok := h.groups[group]
	if !ok {
		// Create a new group and reap timer
		hand = h.factory(group)
		h.groups[group] = hand
		if h.reapDelay != 0 {
			t := time.AfterFunc(h.reapDelay, func() { h.reap(group) })
			h.groupReap[group] = t
		}
	} else {
		// Reset the existing reap timer
		if t, ok := h.groupReap[group]; ok {
			t.Reset(h.reapDelay)
		}
	}

	return hand
}

// reap is called after the reap delay to remove a group handler. Helps
// avoid retaining a large pool of group handlers.
func (h *handler) reap(group string) {
	h.l.Lock()
	defer h.l.Unlock()

	if t, ok := h.groupReap[group]; ok {
		t.Stop()
		delete(h.groupReap, group)
	}
	delete(h.groups, group)
}

// HandlerFactory is a function used to create a new http.Handler for the
// given group name.
type HandlerFactory func(key string) http.Handler

// RequestGrouper is a function used to examine an HTTP request and return a
// key, which is used to group the request to a specific handler.
type RequestGrouper func(r *http.Request) string

// GroupByRequestIP is used to make a best-effort attempt at determining the
// original requestor's IP address. The order of precedence is:
//
// 1. X-Forwarded-For header value.
// 2. IP address derived from the RemoteAddr of the request.
// 3. RemoteAddr raw value of the request (auto-set by the HTTP server).
func GroupByRequestIP(r *http.Request) string {
	// First try the X-Forwarded-For header.
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		forwardedIP := strings.Split(forwardedFor, ",")[0]
		return strings.TrimSpace(forwardedIP)
	}

	// Try the host:port version of the RemoteAddr of the request. This is the
	// default format automatically set by Go's built-in HTTP server.
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && remoteIP != "" {
		return remoteIP
	}

	// Fall back to just returning the raw RemoteAddr; the specific HTTP
	// server may have another format that we can't guess at.
	return r.RemoteAddr
}
