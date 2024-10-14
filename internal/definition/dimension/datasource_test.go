// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dimension

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/metrics_metadata"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestDataSource(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[[]*metrics_metadata.Dimension]{
		{
			Name: "no provider set",
			Meta: func(testing.TB) any {
				return nil
			},
			Resource: NewDataSource(),
			Encoder: func(t *[]*metrics_metadata.Dimension, rd *schema.ResourceData) error {
				return nil
			},
			Decoder: func(_ *schema.ResourceData) (*[]*metrics_metadata.Dimension, error) {
				return nil, nil
			},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "no matching values",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/dimension": func(wr http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(wr).Encode(&metrics_metadata.DimensionQueryResponseModel{
						Count: 0,
					})
				},
			}),
			Resource: NewDataSource(),
			Input:    &[]*metrics_metadata.Dimension{},
			Encoder: func(t *[]*metrics_metadata.Dimension, rd *schema.ResourceData) error {
				return errors.Join(
					rd.Set("query", "service_name:my-awesome-service"),
					rd.Set("limit", 1000),
				)
			},
			Decoder: func(rd *schema.ResourceData) (*[]*metrics_metadata.Dimension, error) {
				return nil, nil
			},
		},
		{
			Name: "matched values",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/dimension": func(wr http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(wr).Encode(&metrics_metadata.DimensionQueryResponseModel{
						Count: 1,
						Results: []*metrics_metadata.Dimension{
							{Value: "my-awesome-service"},
						},
					})
				},
			}),
			Resource: NewDataSource(),
			Input:    &[]*metrics_metadata.Dimension{},
			Encoder: func(t *[]*metrics_metadata.Dimension, rd *schema.ResourceData) error {
				err := errors.Join(
					rd.Set("query", "service_name:my-awesome-*"),
					rd.Set("limit", 1000),
				)
				if err != nil {
					return err
				}
				values := []string{}
				for _, dim := range *t {
					values = append(values, dim.Value)
				}
				return rd.Set("values", values)
			},
			Decoder: func(rd *schema.ResourceData) (*[]*metrics_metadata.Dimension, error) {
				v, ok := rd.GetOk("values")
				if !ok {
					return nil, errors.New("missing value definition")
				}
				data := []*metrics_metadata.Dimension{}
				for _, dim := range v.([]any) {
					data = append(data, &metrics_metadata.Dimension{Value: dim.(string)})
				}

				return &data, nil
			},
			Expect: &[]*metrics_metadata.Dimension{
				{Value: "my-awesome-service"},
			},
		},
		{
			Name: "failed operation",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/dimension": func(wr http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					wr.WriteHeader(http.StatusGatewayTimeout)
					_, _ = io.WriteString(wr, "Unable to connect to downstream service")
				},
			}),
			Resource: NewDataSource(),
			Input:    &[]*metrics_metadata.Dimension{},
			Encoder: func(t *[]*metrics_metadata.Dimension, rd *schema.ResourceData) error {
				return errors.Join(
					rd.Set("query", "service_name:my-awesome-service"),
					rd.Set("limit", 1000),
				)
			},
			Decoder: func(rd *schema.ResourceData) (*[]*metrics_metadata.Dimension, error) {
				return nil, nil
			},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "Bad status 504: Unable to connect to downstream service"},
			},
		},
		{
			Name: "too many results returned",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/dimension": func(wr http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(wr).Encode(&metrics_metadata.DimensionQueryResponseModel{
						Count: 50_000,
					})
				},
			}),
			Resource: NewDataSource(),
			Input:    &[]*metrics_metadata.Dimension{},
			Encoder: func(t *[]*metrics_metadata.Dimension, rd *schema.ResourceData) error {
				return errors.Join(
					rd.Set("query", "service_name:my-awesome-service"),
					rd.Set("limit", 2),
				)
			},
			Decoder: func(rd *schema.ResourceData) (*[]*metrics_metadata.Dimension, error) {
				return nil, nil
			},
			Issues: diag.Diagnostics{
				{
					Severity: diag.Warning,
					Summary:  "Number of matched results exceeds allowed returned limit, values truncated",
					Detail:   "Adjust the query to be more selective or increase the limit to avoid this issue",
				},
			},
		},
	} {
		tc.TestRead(t)
	}
}
