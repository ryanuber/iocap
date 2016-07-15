package iocap

/*
Package iocap provides rate limiting for data streams using the familiar
`io.Reader` and `io.Writer` interfaces.

Rate limits are expressed in bytes per second by calling PerSecond:

	rate := iocap.PerSecond(512*1024) // 512K/s

Readers and Writers are created by passing in an existing io.Reader or
io.Writer along with a rate:

	reader := iocap.NewReader(r, rate)
	writer := iocap.NewWriter(w, rate)

Rate limiting can also be applied to an http.Handler or http.ResponseWriter
using the wrapper methods:

	h := iocap.LimitHTTPHandler(http.FileServer(http.Dir(".")), rate)
	w := iocap.LimitResponseWriter(w, rate)
*/
