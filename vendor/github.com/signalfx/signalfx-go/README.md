# Go client library for SignalFx

Note: This library is experimental. Do not rely on it yet.

This is a programmatic interface in Go for SignalFx's metadata and ingest APIs.

# TODO

* Finish generating models and fixing bugs therein (see Questions below)
* Include APIs for metric reading (signalflow, etc!)

# Example

```
import "github.com/signalfx/signalfx-go"

// The client can be customized by backing options onto the end. Check the
// godoc for more info!

// Instantiate your own client if you want to customize its options
// or test with a RoundTripper
httpClient := &http.Client{…}
client, err := NewClient("your-token-here", HTTPClient(httpClient))
```

# Questions

## Why are the class names sometimes long and the source file names prefixed with `model_`?

The request and response bodies for this library are machine generated from our OpenAPI specs using [OpenAPI code generator](https://github.com/OpenAPITools/openapi-generator). This is a real boon for everyone, keeping the documentation as a source of truth and ensuring that this library has support for all the things!

This means that some of our type names are verbose. It's fine, you only type code once and the benefits are worth it.
