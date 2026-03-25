// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package metadata

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetModuleFunctionMetadata(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		want    *Metadata
		errVal  string
	}{
		{
			name: "successfully retrieves metadata",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()

				meta := Metadata{
					Arguments: []Argument{
						{
							Name:        "arg1",
							Type:        "string",
							Label:       "Argument 1",
							Description: "This is argument 1",
						},
					},
					VizOptions: VizOptions{
						Suffix: "ms",
						Unit:   "milliseconds",
					},
				}
				_ = json.NewEncoder(w).Encode(&meta)
			},
			want: &Metadata{
				Arguments: []Argument{
					{
						Name:        "arg1",
						Type:        "string",
						Label:       "Argument 1",
						Description: "This is argument 1",
					},
				},
				VizOptions: VizOptions{
					Suffix: "ms",
					Unit:   "milliseconds",
				},
			},
		},
		{
			name: "handles non-200 response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad request", http.StatusBadRequest)
			},
			errVal: `GetModuleFunctionMetadata "/v2/signalflow/_/extractMetadata": response: bad request`,
		},
		{
			name: "bad json data sent",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()
				_, _ = w.Write([]byte("text"))
			},
			errVal: `GetModuleFunctionMetadata "/v2/signalflow/_/extractMetadata": json parsing: invalid character 'e' in literal true (expecting 'r')`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tc.handler)
			t.Cleanup(server.Close)

			domain, err := url.Parse(server.URL)
			require.NoError(t, err)

			client := NewClient(domain, "test-token", func(c *Client) {
				c.net = server.Client()
			})

			actual, err := client.GetModuleFunctionMetadata(
				t.Context(),
				"mypackage.internal.platform",
				"detect",
				"latency",
			)

			assert.Equal(t, tc.want, actual)
			if tc.errVal == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.errVal, "Must return expected error")
			}
		})
	}
}
