// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestDataSourcePagerDutyMetadata(t *testing.T) {
	t.Parallel()

	d := NewDataSourcePagerDuty()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)

	assert.Equal(t, "signalfx_pagerduty_integration", resp.TypeName)
}

func TestDataSourcePagerDutySchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fwtest.DataSourceSchemaValidate(NewDataSourcePagerDuty(), dataSourcePagerDutyModel{}))
}

func TestDataSourcePagerDutyRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	implementation := &DataSourcePagerDuty{}
	schemaResponse := &datasource.SchemaResponse{}
	implementation.Schema(ctx, datasource.SchemaRequest{}, schemaResponse)
	response := &datasource.ReadResponse{}

	implementation.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Raw:    tftypes.NewValue(tftypes.Bool, true),
		Schema: schemaResponse.Schema,
	}}, response)

	assert.True(t, response.Diagnostics.HasError())
}

func TestDataSourcePagerDutyMockedRead(t *testing.T) {
	endpoints := map[string]http.Handler{
		"GET /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, url.Values{
				"name": []string{"Primary PagerDuty"},
				"type": []string{"PagerDuty"},
			}, r.URL.Query())

			if err := json.NewEncoder(w).Encode(map[string]any{
				"count": 1,
				"results": []map[string]any{{
					"id":      "pagerduty-id",
					"name":    "Primary PagerDuty",
					"enabled": true,
					"type":    "PagerDuty",
				}},
			}); err != nil {
				t.Errorf("write PagerDuty search response: %v", err)
			}
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t,
			endpoints,
			fwtest.WithMockDataSources(NewDataSourcePagerDuty),
		),
		Steps: []testresource.TestStep{{
			ConfigFile: config.StaticFile("testdata/pagerduty_data_source.tf"),
			Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("data.signalfx_pagerduty_integration.test", "id", "pagerduty-id"),
				testresource.TestCheckResourceAttr("data.signalfx_pagerduty_integration.test", "name", "Primary PagerDuty"),
				testresource.TestCheckResourceAttr("data.signalfx_pagerduty_integration.test", "enabled", "true"),
			),
		}},
	})
}

func TestDataSourcePagerDutyMockedErrors(t *testing.T) {
	for _, test := range []struct {
		name      string
		endpoints map[string]http.Handler
		error     *regexp.Regexp
	}{
		{
			name: "API error",
			endpoints: map[string]http.Handler{
				"GET /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
				}),
			},
			error: regexp.MustCompile(`status code 503`),
		},
		{
			name:  "missing required name",
			error: regexp.MustCompile(`The argument "name" is required`),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			configuration := `data "signalfx_pagerduty_integration" "test" {}`
			if test.endpoints != nil {
				configuration = ""
			}

			step := testresource.TestStep{
				Config:      configuration,
				ConfigFile:  config.StaticFile("testdata/pagerduty_data_source.tf"),
				PlanOnly:    true,
				ExpectError: test.error,
			}
			if configuration != "" {
				step.ConfigFile = nil
			}

			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
					t,
					test.endpoints,
					fwtest.WithMockDataSources(NewDataSourcePagerDuty),
				),
				Steps: []testresource.TestStep{step},
			})
		})
	}
}
