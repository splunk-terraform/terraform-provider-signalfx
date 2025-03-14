// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package orgtoken

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/orgtoken"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

const (
	ResourceName = "signalfx_org_token"
)

func NewResource() *schema.Resource {
	return &schema.Resource{
		SchemaFunc:    newSchema,
		ReadContext:   resourceRead,
		CreateContext: resourceCreate,
		UpdateContext: resourceUpdate,
		DeleteContext: resourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceRead(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	sfx, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	token, err := sfx.GetOrgToken(ctx, data.Id())
	if err != nil {
		return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
	}

	return tfext.AsErrorDiagnostics(encodeTerraform(token, data))
}

func resourceCreate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	sfx, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	details, err := decodeTerraform(data)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	token, err := sfx.CreateOrgToken(ctx, &orgtoken.CreateUpdateTokenRequest{
		Name:          details.Name,
		AuthScopes:    details.AuthScopes,
		Description:   details.Description,
		Limits:        details.Limits,
		Notifications: details.Notifications,
		Disabled:      details.Disabled,
	})

	if err != nil {
		return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
	}

	data.SetId(token.Name)
	return nil
}

func resourceUpdate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	sfx, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	details, err := decodeTerraform(data)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	if data.Id() == "" {
		data.SetId(details.Name)
	}

	token, err := sfx.UpdateOrgToken(ctx, data.Id(), &orgtoken.CreateUpdateTokenRequest{
		Name:          details.Name,
		AuthScopes:    details.AuthScopes,
		Description:   details.Description,
		Limits:        details.Limits,
		Notifications: details.Notifications,
		Disabled:      details.Disabled,
	})
	if err != nil {
		return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
	}

	return tfext.AsErrorDiagnostics(encodeTerraform(token, data))
}

func resourceDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	sfx, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	if data.Id() == "" {
		details, err := decodeTerraform(data)
		if err != nil {
			return tfext.AsErrorDiagnostics(err)
		}
		data.SetId(details.Name)
	}

	err = sfx.DeleteOrgToken(ctx, data.Id())
	if err == nil {
		data.SetId("")
	}

	return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
}
