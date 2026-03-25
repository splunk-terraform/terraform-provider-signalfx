// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package experimental

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/metadata"
)

func TestNewInspector(t *testing.T) {
	t.Parallel()

	domain, err := url.Parse("https://api.example.com")
	require.NoError(t, err)

	actual := NewInspector(domain, "token")

	require.NotNil(t, actual)
	assert.NotNil(t, actual.meta)
	assert.NotNil(t, actual.flow)
}

func TestInspectorGetAutoDetectorInputs(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name               string
		graphStatusCode    int
		graphBody          string
		metadataStatusCode int
		metadataBody       string
		want               []metadata.Argument
		errVal             string
	}{
		{
			name:            "flow endpoint error is returned",
			graphStatusCode: http.StatusServiceUnavailable,
			graphBody:       "Service Unavailable",
			errVal:          "unable to load exec graph: GetExecutionGraph \"/v2/signalflow/_/getSignalFlowModel\": unexpected status code: 503: Service Unavailable",
		},
		{
			name:               "metadata endpoint error is returned",
			graphStatusCode:    http.StatusOK,
			graphBody:          `[{"module":"signalfx.detectors.autodetect.apm","name":"requests","type":"IMPORT"},{"start":{"functionName":"requests.blended"},"type":"DETECT"}]`,
			metadataStatusCode: http.StatusBadRequest,
			metadataBody:       "bad request",
			errVal:             "unable to get module function metadata: GetModuleFunctionMetadata \"/v2/signalflow/_/extractMetadata\": response: bad request",
		},
		{
			name:               "successful response flattens metadata arguments",
			graphStatusCode:    http.StatusOK,
			graphBody:          `[{"module":"signalfx.detectors.autodetect.apm","name":"requests","type":"IMPORT"},{"start":{"functionName":"requests.blended"},"type":"DETECT"}]`,
			metadataStatusCode: http.StatusOK,
			metadataBody: `{"arguments":[
				{"argName":"service","argType":"string","description":"Service name","defaultValue":"checkout","unit":"count","metric":"sf_metric"},
				{"argName":"window","argType":"int","defaultValue":5}
			]}`,
			want: []metadata.Argument{
				{
					Name:         "service",
					Type:         "string",
					Description:  "Service name",
					DefaultValue: "checkout",
					Unit:         "count",
					Metric:       "sf_metric",
				},
				{
					Name:         "window",
					Type:         "int",
					DefaultValue: float64(5),
				},
			},
		},
		{
			name:               "optional fields are omitted when empty",
			graphStatusCode:    http.StatusOK,
			graphBody:          `[{"module":"signalfx.detectors.autodetect.apm","name":"requests","type":"IMPORT"},{"start":{"functionName":"requests.blended"},"type":"DETECT"}]`,
			metadataStatusCode: http.StatusOK,
			metadataBody:       `{"arguments":[{"argName":"service","argType":"string"}]}`,
			want: []metadata.Argument{
				{
					Name: "service",
					Type: "string",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/signalflow/_/getSignalFlowModel":
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "true", r.URL.Query().Get("parseDetectors"))
					w.WriteHeader(tc.graphStatusCode)
					_, _ = w.Write([]byte(tc.graphBody))
				case "/v2/signalflow/_/extractMetadata":
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, "signalfx.detectors.autodetect.apm", r.URL.Query().Get("modulePath"))
					assert.Equal(t, "requests", r.URL.Query().Get("moduleName"))
					assert.Equal(t, "blended", r.URL.Query().Get("functionName"))
					w.WriteHeader(tc.metadataStatusCode)
					_, _ = w.Write([]byte(tc.metadataBody))
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)

			domain, err := url.Parse(server.URL)
			require.NoError(t, err)

			inspect := NewInspector(domain, "token")

			actual, _, err := inspect.GetAutoDetectorArgumentsAndFilters(t.Context(), "detect program text")

			assert.Equal(t, tc.want, actual)
			if tc.errVal == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.errVal)
			}
		})
	}
}
