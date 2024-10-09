// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/signalfx/signalfx-go"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

// NewTestHTTPMockMeta allows for a `providermeta.Meta` to be configured as if it would be set by the provider.
// The underlying mux uses the [http.ServerMux] using `HandlerFunc` method which allows setting the expected
// method on routes, for example:
//
//	routes := map[string]http.HandlerFunc{
//		"GET /v2/<resource>": func(...){ ... } // Only accepts GET method request matching the route
//		"/v2/login": func(...){ ... }          // Handles all request for `/v2/login` regardless of method
//	}
func NewTestHTTPMockMeta(routes map[string]http.HandlerFunc) func(testing.TB) any {
	mux := http.NewServeMux()
	for pattern, handler := range routes {
		mux.HandleFunc(pattern, handler)
	}

	return func(t testing.TB) any {
		s := httptest.NewServer(mux)
		// Ensure that the server is correctly shutdown
		// once the test is done to avoid leaking resources
		t.Cleanup(s.Close)

		sfx, _ := signalfx.NewClient(
			t.Name(),
			signalfx.HTTPClient(s.Client()),
			signalfx.APIUrl(s.URL),
		)

		return &pmeta.Meta{
			APIURL:       s.URL,
			CustomAppURL: s.URL,
			Client:       sfx,
		}
	}
}
