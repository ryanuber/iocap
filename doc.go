/*
Package iocap provides rate limiting for data streams using the familiar
io.Reader and io.Writer interfaces.

Rate limits are expressed in bytes per second by calling PerSecond:

	rate := iocap.PerSecond(512*1024) // 512K/s

Readers and Writers are created by passing in an existing io.Reader or
io.Writer along with a rate.

	r = iocap.NewReader(r, rate)
	w = iocap.NewWriter(w, rate)

Rate limits can be applied to multiple readers and/or writers by creating
a rate limiting group for them.

	// Create the shared group
	g := iocap.NewGroup(rate)

	// Pull a new reader and writer off of the group. Rate limits are now
	// applied and enforced across both.
	r = g.NewReader(r)
	w = g.NewWriter(w)
*/
package iocap
