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

// Handler creates a new rate limited HTTP handler wrapper. The rate described
// by ro is used to rate limit each request independently.
func Handler(h http.Handler, ro iocap.RateOpts) http.Handler {
	return &handler{
		h:    h,
		opts: ro,
	}
}

// GroupHandler is like Handler, but wraps an http.Handler with group rate
// limiting such that all requests share the same quota.
func GroupHandler(h http.Handler, g *iocap.Group) http.Handler {
	return &handler{
		h:     h,
		group: g,
	}
}

// ServeHTTP implements the http.Handler interface, writing responses using
// a rate limited response writer.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.group != nil {
		w = &responseWriter{
			writer:         h.group.NewWriter(w),
			ResponseWriter: w,
		}
	} else {
		w = &responseWriter{
			writer:         iocap.NewWriter(w, h.opts),
			ResponseWriter: w,
		}
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

// Write implements part of the http.ResponseWriter interface, calling the
// underlying rate limited writer instead of directly writing out bytes.
func (w *responseWriter) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}
