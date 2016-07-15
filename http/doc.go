/*
Package httpcap provides rate limiting for HTTP handlers and response writers.

Rate limiting can be applied to an http.Handler either on a per-request
basis, or using a rate limiting group.

	h = httpcap.Handler(h, rate)
	...
	g := iocap.NewGroup(rate)
	h = httpcap.HandlerGroup(h, g)

Rate limiting can be applied to an http.ResponseWriter in the same manner.

	w = httpcap.ResponseWriter(w, rate)
	...
	g := iocap.NewGroup(rate)
	w = httpcap.ResponseWriterGroup(w, g)
*/
package httpcap
