[![Build Status](https://travis-ci.org/ryanuber/iocap.svg)][travis]
[![GoDoc](https://godoc.org/github.com/ryanuber/iocap?status.svg)][godoc]

iocap
=====

Go package for rate limiting data streams using the familiar `io.Reader` and
`io.Writer` interfaces.

`iocap` provides simple wrappers over arbitrary `io.Reader` and `io.Writer`
instances to allow throttling throughput of either read or write operations.
Data streams can be rate limited individually or grouped together to provide an
aggregate rate over multiple operations.

For examples and usage, see the [godoc][].

## How it works

Under the hood, `iocap` uses a very simple [leaky bucket][] implementation to
shape the flow of traffic. This implementation uses timestamps instead of a
constant-rate "leak" to empty the bucket. The reason for this is to allow
readers and writers to utilize the leaky bucket without requiring additional
setup, including starting/stopping of timers or other goroutines.

[travis]: https://travis-ci.org/ryanuber/iocap
[godoc]: https://godoc.org/github.com/ryanuber/iocap
[leaky bucket]: https://en.wikipedia.org/wiki/Leaky_bucket
