package httpcap

import (
	"net/http"

	"github.com/ryanuber/iocap"
)

// handler is a wrapper over a normal http.Handler, allowing the
// rate to be controlled while sending data back to clients.
type handler struct {
	h     http.Handler
	opts  iocap.RateOpts
	group *iocap.Group
}

// Handler creates a new rate limited HTTP handler wrapper. Each request
// is independently rate limited according to opts.
func Handler(h http.Handler, opts iocap.RateOpts) http.Handler {
	return &handler{
		h:    h,
		opts: opts,
	}
}

// HandlerGroup creates a new rate limited HTTP handler constrained
// by the given rate limiting group. All requests will share the rate limit
// held by g.
func HandlerGroup(h http.Handler, g *iocap.Group) http.Handler {
	return &handler{
		h:     h,
		group: g,
	}
}

// ServeHTTP implements the http.Handler interface, writing responses using
// a rate limited response writer.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.group != nil {
		w = ResponseWriterGroup(w, h.group)
	} else {
		w = ResponseWriter(w, h.opts)
	}
	h.h.ServeHTTP(w, r)
}

// responseWriter wraps an http.ResponseWriter in a rate limited
// writer, effectively throttling throughput from the HTTP server to
// all of its clients.
type responseWriter struct {
	writer *iocap.Writer
	http.ResponseWriter
}

// ResponseWriter creates a new rate limited, wrapped response writer.
func ResponseWriter(w http.ResponseWriter, opts iocap.RateOpts) http.ResponseWriter {
	return &responseWriter{
		writer:         iocap.NewWriter(w, opts),
		ResponseWriter: w,
	}
}

// ResponseWriterGroup returns a new rate limited http.ResponseWriter
// constrained by the group g.
func ResponseWriterGroup(w http.ResponseWriter, g *iocap.Group) http.ResponseWriter {
	return &responseWriter{
		writer:         g.NewWriter(w),
		ResponseWriter: w,
	}
}

// Write implements part of the http.ResponseWriter interface, calling the
// underlying rate limited writer instead of directly writing out bytes.
func (w *responseWriter) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}
