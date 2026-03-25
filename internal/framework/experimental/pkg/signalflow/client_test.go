// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package flow

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewClient(&url.URL{}, "token"))
	assert.NotNil(t, NewClient(&url.URL{}, "token", func(c *Client) {
		c.net = nil
	}))
}

func TestClientGetExecutionGraph(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		expect  ExecutionGraph
		errVal  string
	}{
		{
			name: "endpoint offline",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			},
			errVal: "GetExecutionGraph \"/v2/signalflow/_/getSignalFlowModel\": unexpected status code: 503: Service Unavailable",
		},
		{
			name: "invalid json returned",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("invalid json"))
			},
			errVal: "invalid character 'i' looking for beginning of value",
		},
		{
			name: "valid response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				data := ExecutionGraph{
					ExecutionBlockStream{
						Typed: "PLOT",
						Start: ExecutionBlockStreamMethod{
							OriginalText: "data('my-metric')",
							FunctionName: "data",
							Arguments: ExecutionBlockArguments{
								"metric": &ExecutionBlockArgumentLiteral{
									Type:  "string",
									Value: "my-metric",
								},
							},
						},
						Methods: []ExecutionBlockStreamMethod{
							{
								OriginalText: ".publish('my-metric')",
								FunctionName: "publish",
								Arguments: ExecutionBlockArguments{
									"label": &ExecutionBlockArgumentLiteral{
										Type:  "string",
										Value: "my-metric",
									},
								},
							},
						},
					},
				}
				_ = json.NewEncoder(w).Encode(data)
			},
			expect: ExecutionGraph{
				ExecutionBlockStream{
					Typed: "PLOT",
					Start: ExecutionBlockStreamMethod{
						OriginalText: "data('my-metric')",
						FunctionName: "data",
						Arguments: ExecutionBlockArguments{
							"metric": &ExecutionBlockArgumentLiteral{
								Type:  "string",
								Value: "my-metric",
							},
						},
					},
					Methods: []ExecutionBlockStreamMethod{
						{
							OriginalText: ".publish('my-metric')",
							FunctionName: "publish",
							Arguments: ExecutionBlockArguments{
								"label": &ExecutionBlockArgumentLiteral{
									Type:  "string",
									Value: "my-metric",
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tc.handler)
			t.Cleanup(server.Close)

			u, _ := url.Parse(server.URL)

			client := NewClient(u, "token", func(c *Client) {
				c.net = server.Client()
			})

			graph, err := client.GetExecutionGraph(
				t.Context(),
				"A = data('my-metric').publish('my-metric')",
			)
			assert.Equal(t, tc.expect, graph, "Must match the expected value")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error")
			} else {
				assert.NoError(t, err, "Must not return an error")
			}
		})
	}
}
