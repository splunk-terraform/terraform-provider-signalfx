// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/integration"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceSplunkOncallMetadata(t *testing.T) {
	t.Parallel()

	r := NewResourceSplunkOncall()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)

	assert.Equal(t, "signalfx_integration_splunk_oncall", resp.TypeName)
}

func TestResourceSplunkOnCallSchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fwtest.ResourceSchemaValidate(NewResourceSplunkOncall(), resourceSplunkOnCallModel{}))
}

func TestResourceSplunkOncallUnitTest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		endpoints map[string]http.Handler
		cases     []testresource.TestStep
	}{
		{
			name: "Correctly configured client",
			endpoints: map[string]http.Handler{
				"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var data integration.VictorOpsIntegration
					if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					if data.Name == "" || data.PostUrl == "" {
						http.Error(w, "name and post_url are required", http.StatusBadRequest)
						return
					}

					data.Id = "test-id"
					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"GET /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					data := integration.VictorOpsIntegration{
						Id:      "test-id",
						Name:    "Test Integration",
						Enabled: true,
						PostUrl: "https://example.com/post",
					}
					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
			},
			cases: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_splunk_oncall.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx.signalfx_integration_splunk_oncall.test", "id", "test-id"),
						testresource.TestCheckResourceAttr("signalfx.signalfx_integration_splunk_oncall.test", "enabled", "true"),
						testresource.TestCheckResourceAttr("signalfx.signalfx_integration_splunk_oncall.test", "name", "Test Integration"),
						testresource.TestCheckResourceAttr("signalfx.signalfx_integration_splunk_oncall.test", "post_url", "https://example.com/post"),
					),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testresource.Test(
				t,
				testresource.TestCase{
					IsUnitTest: true,
					TerraformVersionChecks: []tfversion.TerraformVersionCheck{
						tfversion.RequireAbove(tfversion.Version0_12_26),
					},
					ProtoV5ProviderFactories: fwtest.NewMockProviderFactory(
						t,
						tc.endpoints,
						fwtest.WithMockResources(NewResourceSplunkOncall),
					),
					Steps: tc.cases,
				},
			)
		})
	}
}
