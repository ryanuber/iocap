/*
Package httpcap provides rate limiting for HTTP handlers and response writers.

Rate limiting can be applied to an http.Handler either on a per-request
basis, or using a rate limiting group.

	h = httpcap.Handler(h, rate)
	...
	g := iocap.NewGroup(rate)
	h = httpcap.GroupHandler(h, g)

See the LimitByRequestIP method for a short-hand quick start.
*/
package httpcap
