/*
Package mapper provides HTTP request mapping based on arbitrary criteria,
including the raw HTTP request details. This enables servicing requests
with specialized handlers which are potentially created just in time to
handle the request.

The following example shows how this is useful with the iocap packages.

	// Group requests by the request IP using the built-in helper.
	groupFn := mapper.GroupByRequestIP

	// Create the base HTTP handler.
	hand := http.FileServer(http.Dir("."))

	// Create an HTTP handler factory that will create new rate-limited
	// groups for our handler.
	handFn := func(_ string) http.Handler {
		rate := iocap.PerSecond(512 * 1024)
		group := iocap.NewGroup(rate)
		return httpcap.HandlerGroup(hand, group)
	}

	// Set group expiration to 1 hour. This allows keeping rate limit
	// groups around for some time while mitigating infinite memory growth
	// as the server observes new clients.
	reapTime := time.Hour

	// Create the mapper and start mapping in requests.
	rm := mapper.New(groupFn, handFn, reapTime)
	http.ListenAndServe(":8080", rm)
*/
package mapper
