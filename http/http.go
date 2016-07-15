package httpcap

import (
	"net/http"

	"github.com/ryanuber/iocap"
)

// LimitedHandler is a wrapper over a normal http.Handler, allowing the
// rate to be controlled while sending data back to clients.
type LimitedHandler struct {
	h     http.Handler
	opts  iocap.RateOpts
	group *iocap.Group
}

// LimitHandler creates a new rate limited HTTP handler wrapper.
func LimitHandler(h http.Handler, opts iocap.RateOpts) *LimitedHandler {
	return &LimitedHandler{
		h:    h,
		opts: opts,
	}
}

// LimitHandlerGroup creates a new rate limited HTTP handler constrained
// by the given rate limiting group. All requests will share the rate limit
// held by g.
func LimitHandlerGroup(h http.Handler, g *iocap.Group) *LimitedHandler {
	return &LimitedHandler{
		h:     h,
		group: g,
	}
}

// ServeHTTP implements the http.Handler interface, writing responses using
// a rate limited response writer.
func (h *LimitedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.group != nil {
		w = LimitResponseWriterGroup(w, h.group)
	} else {
		w = LimitResponseWriter(w, h.opts)
	}
	h.h.ServeHTTP(w, r)
}

// LimitedResponseWriter wraps an http.ResponseWriter in a rate limited
// writer, effectively throttling throughput from the HTTP server to
// all of its clients.
type LimitedResponseWriter struct {
	writer *iocap.Writer
	http.ResponseWriter
}

// LimitResponseWriter creates a new rate limited, wrapped response writer.
func LimitResponseWriter(w http.ResponseWriter, opts iocap.RateOpts) *LimitedResponseWriter {
	return &LimitedResponseWriter{
		writer:         iocap.NewWriter(w, opts),
		ResponseWriter: w,
	}
}

// LimitResponseWriterGroup returns a new rate limited http.ResponseWriter
// constrained by the group g.
func LimitResponseWriterGroup(w http.ResponseWriter, g *iocap.Group) *LimitedResponseWriter {
	return &LimitedResponseWriter{
		writer:         g.NewWriter(w),
		ResponseWriter: w,
	}
}

// Write implements part of the http.ResponseWriter interface, calling the
// underlying rate limited writer instead of directly writing out bytes.
func (w *LimitedResponseWriter) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}
