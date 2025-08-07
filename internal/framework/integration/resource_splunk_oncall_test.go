// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
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
	}{} {
		t.Run(tc.name, func(t *testing.T) {
			testresource.Test(t,
				testresource.TestCase{
					IsUnitTest: true,
					Steps:      tc.cases,
					TerraformVersionChecks: []tfversion.TerraformVersionCheck{
						tfversion.RequireAbove(tfversion.Version0_12_26),
					},
					ProtoV5ProviderFactories: fwtest.NewMockProviderFactory(
						t,
						tc.endpoints,
						fwtest.WithMockResources(NewResourceSplunkOncall),
					),
				},
			)
		})
	}
}
