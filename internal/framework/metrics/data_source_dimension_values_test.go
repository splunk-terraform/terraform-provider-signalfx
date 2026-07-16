// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwmetrics

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/signalfx/signalfx-go/metrics_metadata"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestDataSourceDimensionValuesMetadataAndSchema(t *testing.T) {
	t.Parallel()
	implementation := NewDataSourceDimensionValues()
	metadata := &datasource.MetadataResponse{}
	implementation.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_dimension_values", metadata.TypeName)
	assert.NoError(t, fwtest.DataSourceSchemaValidate(implementation, dataSourceDimensionValuesModel{
		Values: types.ListNull(types.StringType),
	}))

	response := &datasource.SchemaResponse{}
	implementation.Schema(context.Background(), datasource.SchemaRequest{}, response)
	assert.Len(t, response.Schema.Attributes["limit"].(schema.Int64Attribute).Validators, 1)
}

func TestDataSourceDimensionValuesRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &DataSourceDimensionValues{}
	schemaResponse := &datasource.SchemaResponse{}
	implementation.Schema(ctx, datasource.SchemaRequest{}, schemaResponse)
	response := &datasource.ReadResponse{}
	implementation.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Bool, true), Schema: schemaResponse.Schema,
	}}, response)
	assert.True(t, response.Diagnostics.HasError())
}

func TestDataSourceDimensionValuesMockedPagination(t *testing.T) {
	var mu sync.Mutex
	calls := make(map[string]int)
	endpoints := map[string]http.Handler{
		"GET /v2/dimension": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Equal(t, "key:host", query.Get("query"))
			assert.Equal(t, "value", query.Get("orderBy"))
			limit, err := strconv.Atoi(query.Get("limit"))
			if err != nil {
				t.Errorf("parse limit: %v", err)
				return
			}
			offset, err := strconv.Atoi(query.Get("offset"))
			if err != nil {
				t.Errorf("parse offset: %v", err)
				return
			}
			mu.Lock()
			calls[strconv.Itoa(offset)+":"+strconv.Itoa(limit)]++
			mu.Unlock()

			response := &metrics_metadata.DimensionQueryResponseModel{Count: 1002}
			switch offset {
			case 0:
				response.Results = make([]*metrics_metadata.Dimension, 1000)
				for index := range response.Results {
					response.Results[index] = &metrics_metadata.Dimension{Value: "host-" + strconv.Itoa(index)}
				}
			case 1000:
				response.Results = []*metrics_metadata.Dimension{{Value: "host-1000"}}
			default:
				http.Error(w, "unexpected offset", http.StatusBadRequest)
				return
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Errorf("write dimension response: %v", err)
			}
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockDataSources(NewDataSourceDimensionValues)),
		Steps: []testresource.TestStep{{
			Config: `data "signalfx_dimension_values" "test" {
  query    = "key:host"
  order_by = "value"
  limit    = 1001
}`,
			Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("data.signalfx_dimension_values.test", "id", "key:host"),
				testresource.TestCheckResourceAttr("data.signalfx_dimension_values.test", "values.#", "1001"),
				testresource.TestCheckResourceAttr("data.signalfx_dimension_values.test", "values.0", "host-0"),
				testresource.TestCheckResourceAttr("data.signalfx_dimension_values.test", "values.1000", "host-1000"),
			),
		}},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.NotEmpty(t, calls["0:1000"])
	assert.NotEmpty(t, calls["1000:1"])
}

func TestDataSourceDimensionValuesZeroLimit(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, nil, fwtest.WithMockDataSources(NewDataSourceDimensionValues)),
		Steps: []testresource.TestStep{{
			Config: `data "signalfx_dimension_values" "test" {
  query = "key:host"
  limit = 0
}`,
			Check: testresource.TestCheckResourceAttr("data.signalfx_dimension_values.test", "values.#", "0"),
		}},
	})
}

func TestDataSourceDimensionValuesErrors(t *testing.T) {
	for _, test := range []struct {
		name      string
		config    string
		endpoints map[string]http.Handler
		error     *regexp.Regexp
	}{
		{
			name: "API error", config: `data "signalfx_dimension_values" "test" { query = "key:host" }`,
			endpoints: map[string]http.Handler{"GET /v2/dimension": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "failed", http.StatusBadGateway)
			})}, error: regexp.MustCompile(`status code 502`),
		},
		{name: "missing query", config: `data "signalfx_dimension_values" "test" {}`, error: regexp.MustCompile(`argument "query" is required`)},
		{name: "negative limit", config: `data "signalfx_dimension_values" "test" {
  query = "key:host"
  limit = -1
}`, error: regexp.MustCompile(`between 0 and 10000`)},
		{name: "excessive limit", config: `data "signalfx_dimension_values" "test" {
  query = "key:host"
  limit = 10001
}`, error: regexp.MustCompile(`between 0 and 10000`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, test.endpoints, fwtest.WithMockDataSources(NewDataSourceDimensionValues)),
				Steps:                    []testresource.TestStep{{Config: test.config, PlanOnly: true, ExpectError: test.error}},
			})
		})
	}
}
