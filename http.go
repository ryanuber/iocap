package iocap

import "net/http"

// LimitedHTTPHandler is a wrapper over a normal http.Handler, allowing the
// rate to be controlled while sending data back to clients.
type LimitedHTTPHandler struct {
	h    http.Handler
	opts RateOpts
}

// LimitHTTPHandler creates a new rate limited HTTP handler wrapper.
func LimitHTTPHandler(h http.Handler, opts RateOpts) *LimitedHTTPHandler {
	return &LimitedHTTPHandler{
		h:    h,
		opts: opts,
	}
}

// ServeHTTP implements the http.Handler interface, writing responses using
// a rate limited response writer.
func (h *LimitedHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.h.ServeHTTP(LimitResponseWriter(w, h.opts), r)
}

// LimitedResponseWriter wraps an http.ResponseWriter in a rate limited
// writer, effectively throttling throughput from the HTTP server to
// all of its clients.
type LimitedResponseWriter struct {
	writer *Writer
	http.ResponseWriter
}

// LimitResponseWriter creates a new rate limited, wrapped response writer.
func LimitResponseWriter(w http.ResponseWriter, opts RateOpts) *LimitedResponseWriter {
	return &LimitedResponseWriter{
		writer:         NewWriter(w, opts),
		ResponseWriter: w,
	}
}

// Write implements part of the http.ResponseWriter interface, calling the
// underlying rate limited writer instead of directly writing out bytes.
func (w *LimitedResponseWriter) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}
