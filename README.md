[![Build Status](https://travis-ci.org/ryanuber/iocap.svg)](https://travis-ci.org/ryanuber/iocap)
[![GoDoc](https://godoc.org/github.com/ryanuber/iocap?status.svg)](https://godoc.org/github.com/ryanuber/iocap)

iocap
=====

Go package for rate limiting data streams using the familiar `io.Reader` and
`io.Writer` interfaces.

`iocap` provides simple wrappers over arbitrary `io.Reader` and `io.Writer`
instances to allow throttling throughput of either read or write operations.

## Features

* Rate limit any `io.Reader` or `io.Writer`
* Rate limit any `http.Handler` or `http.ResponseWriter`

## How it works

Under the hood, `iocap` uses a very simple [leaky bucket][] implementation to
shape the flow of traffic. This implementation uses timestamps instead of a
constant-rate "leak" to empty the bucket. The reason for this is to allow
readers and writers to utilize the leaky bucket without requiring additional
setup, including starting/stopping of timers or other goroutines.

[leaky bucket]: https://en.wikipedia.org/wiki/Leaky_bucket

## Examples

### Reader/Writer

```go
package main

import (
    "bytes"
    "github.com/ryanuber/iocap"
)

func main() {
    data := "helloworld"
    rate := iocap.PerSecond(2) // Whopping 2B/s

    buf := new(bytes.Buffer)
    out := make([]byte, len(data))

    // Write
    w := iocap.NewWriter(buf, rate)
    if _, err := w.Write([]byte(data)); err != nil {
        panic(err)
    }

    // Read
    r := iocap.NewReader(buf, rate)
    if _, err := r.Read(out); err != nil {
        panic(err)
    }

    println("Result:", string(out))
}
```

### HTTP

The following program will start an HTTP server and serve files out of the
current working directory, rate limiting each request to 128K/s.

```go
package main

import (
    "github.com/ryanuber/iocap"
    "net/http"
)

func main() {
    handler := http.FileServer(http.Dir("."))
    rate := iocap.PerSecond(128 * 1024) // 128K/s
    http.ListenAndServe(":8080", iocap.LimitHTTPHandler(handler, rate))
}
```
