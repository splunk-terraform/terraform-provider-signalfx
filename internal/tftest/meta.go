// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
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

// newAcceptanceConfigure is used for acceptance testing purposes and not be used within unit tests.
func newAcceptanceConfigure(context.Context, *schema.ResourceData) (any, diag.Diagnostics) {
	meta := &pmeta.Meta{
		APIURL:       os.Getenv("SFX_API_URL"),
		AuthToken:    os.Getenv("SFX_AUTH_TOKEN"),
		CustomAppURL: "https://app.signalfx.com",
	}

	meta.Client, _ = signalfx.NewClient(
		meta.AuthToken,
		signalfx.APIUrl(meta.APIURL),
	)

	if err := meta.Validate(); err != nil {
		return nil, tfext.AsErrorDiagnostics(err)
	}
	return meta, nil
}
