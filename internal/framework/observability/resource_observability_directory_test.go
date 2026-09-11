// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/directory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceObservabilityDirectoryMetadataAndSchema(t *testing.T) {
	t.Parallel()

	r := NewResourceObservabilityDirectory()
	var metadata resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, &metadata)
	assert.Equal(t, "signalfx_observability_directory", metadata.TypeName)
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, observabilityDirectoryModel{}))
}

func TestObservabilityDirectoryPatchPreservesMembership(t *testing.T) {
	patch := observabilityDirectoryPatch(types.BoolValue(true))
	require.NotNil(t, patch)
	assert.Nil(t, patch.Templates)
	require.NotNil(t, patch.Pinned)
	assert.True(t, *patch.Pinned)

	model := observabilityDirectoryModelFromEntry(&directory.Entry{Path: "team/dashboards", Pinned: true})
	assert.Equal(t, "team/dashboards", model.ID.ValueString())
	assert.True(t, model.Pinned.ValueBool())
}

func TestObservabilityReservedDirectoryPath(t *testing.T) {
	for _, path := range []string{"~templates", "~users", "team/~users", "~observability/homepage"} {
		assert.Equal(t, path, observabilityReservedDirectoryPath(path))
	}
	assert.Empty(t, observabilityReservedDirectoryPath("team/dashboards"))
}

func TestResourceObservabilityDirectoryUnitTest(t *testing.T) {
	store := newDirectoryAPIStore()

	testresource.UnitTest(
		t,
		testresource.TestCase{
			IsUnitTest: true,
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.RequireAbove(tfversion.Version0_12_26),
			},
			ProtoV5ProviderFactories: fwtest.NewMockProto5Server(
				t,
				store.handlers(),
				fwtest.WithMockResources(NewResourceObservabilityDirectory),
			),
			Steps: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_observability_directory.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "id", "teams/platform/dashboards"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "path", "teams/platform/dashboards"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "pinned", "true"),
					),
				},
				{
					ConfigFile: config.StaticFile("testdata/01_observability_directory_updated.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "id", "teams/platform/dashboards"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "pinned", "false"),
					),
				},
			},
		},
	)
}
