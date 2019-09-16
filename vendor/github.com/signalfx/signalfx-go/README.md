# Go client library for SignalFx

[![GoDoc](https://godoc.org/github.com/signalfx/signalfx-go?status.svg)](https://godoc.org/github.com/signalfx/signalfx-go)

This is a programmatic interface in Go for SignalFx's metadata and ingest APIs.

# SignalFlow

There is an **experimental** SignalFlow client in the `signalflow` directory.  An
example of its use is in [signalflow/example].  For full documentation see the
[godocs](https://godoc.org/github.com/signalfx/signalfx-go/signalflow).

# Example

```
import "github.com/signalfx/signalfx-go"

// The client can be customized by backing options onto the end. Check the
// godoc for more info!

// Instantiate your own client if you want to customize its options
// or test with a RoundTripper
httpClient := &http.Client{…}
client := signalfx.NewClient("your-token-here", HTTPClient(httpClient))

// Then do things!
chart, err := client.GetChart("abc123IdHere")
```

# Questions

## Why are there some things missing?

We're working on it, feel free to file an issue if an endpoint is missing!

## Why are the class names sometimes long and the source file names prefixed with `model_`?

The request and response bodies for this library are machine generated from our OpenAPI specs using [OpenAPI code generator](https://github.com/OpenAPITools/openapi-generator). This is a real boon for everyone, keeping the documentation as a source of truth and ensuring that this library has support for all the things!

This means that some of our type names are verbose. It's fine, you only type code once and the benefits are worth it.
