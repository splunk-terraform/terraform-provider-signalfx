// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwembed

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// ResourceIDImporter is an embedable type that will
// enable the resource to be imported by using the provided ID to fetch from the API.
// It implements the additional method required by [resource.ResourceWithImportState].
type ResourceIDImporter struct{}

func (ResourceIDImporter) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
