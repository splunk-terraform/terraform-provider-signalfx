package autoarchivesettings

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	automated_archival "github.com/signalfx/signalfx-go/automated-archival"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

const (
	ResourceName = "signalfx_automated_archival_settings"
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

	settings, err := sfx.GetSettings(ctx)
	if err != nil {
		return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
	}

	return tfext.AsErrorDiagnostics(encodeTerraform(settings, data))
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

	setting, err := sfx.CreateSettings(ctx, &automated_archival.AutomatedArchivalSettings{
		Creator:        details.Creator,
		Created:        details.Created,
		LastUpdatedBy:  details.LastUpdatedBy,
		LastUpdated:    details.LastUpdated,
		Version:        details.Version,
		Enabled:        details.Enabled,
		LookbackPeriod: details.LookbackPeriod,
		GracePeriod:    details.GracePeriod,
		RulesetLimit:   details.RulesetLimit,
		OrgId:          details.OrgId,
	})

	if err != nil {
		return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
	}

	data.SetId(strconv.FormatInt(setting.Version, 10))
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

	setting, err := sfx.UpdateSettings(ctx, &automated_archival.AutomatedArchivalSettings{
		Creator:        details.Creator,
		Created:        details.Created,
		LastUpdatedBy:  details.LastUpdatedBy,
		LastUpdated:    details.LastUpdated,
		Version:        details.Version,
		Enabled:        details.Enabled,
		LookbackPeriod: details.LookbackPeriod,
		GracePeriod:    details.GracePeriod,
		RulesetLimit:   details.RulesetLimit,
		OrgId:          details.OrgId,
	})
	if err != nil {
		return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
	}

	return tfext.AsErrorDiagnostics(encodeTerraform(setting, data))
}

func resourceDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	sfx, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	version, err := strconv.ParseInt(data.State().ID, 10, 64)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}
	deleteSettingsRequest := automated_archival.AutomatedArchivalSettingsDeleteRequest{
		Version: &version,
	}
	err = sfx.DeleteSettings(ctx, &deleteSettingsRequest)
	if err == nil {
		data.SetId("")
	}

	return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
}
