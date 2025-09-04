// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwchart

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/config"
	resourcetest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
	"github.com/stretchr/testify/assert"
)

func TestNewResourceChart(t *testing.T) {
	t.Parallel()

	rc := NewResourceChart()
	assert.NotNil(t, rc, "Must have a valid chart returned")
	var resp resource.MetadataResponse
	rc.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, &resp)
	assert.Equal(t, "signalfx_chart", resp.TypeName, "Must have a valid type name")
}

func TestResourceChart(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		endpoints map[string]http.Handler
		cases     []resourcetest.TestStep
	}{
		{
			name:      "heatmap",
			endpoints: map[string]http.Handler{},
			cases: []resourcetest.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/heatmap.tf"),
					PlanOnly:   true,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resourcetest.UnitTest(
				t,
				resourcetest.TestCase{
					TerraformVersionChecks: []tfversion.TerraformVersionCheck{
						tfversion.RequireAbove(tfversion.Version1_6_0),
					},
					ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, tc.endpoints, fwtest.WithMockResources(NewResourceChart)),
					Steps:                    tc.cases,
				},
			)
		})
	}
}
