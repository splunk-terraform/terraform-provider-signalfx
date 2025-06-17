package autoarchivesettings

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	autoarch "github.com/signalfx/signalfx-go/automated-archival"
	"go.uber.org/multierr"
)

func newSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"creator": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "ID of the creator of the automated archival setting",
		},
		"last_updated_by": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "ID of user who last updated the automated archival setting",
		},
		"created": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "Timestamp of when the automated archival setting was created",
		},
		"last_updated": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "Timestamp of when the automated archival setting was last updated",
		},
		"version": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "Version of the automated archival setting",
		},
		"enabled": {
			Type:        schema.TypeBool,
			Required:    true,
			Description: "Whether the automated archival is enabled for this organization or not",
		},
		"lookback_period": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "This tracks if a metric was unused in the past N number of days (N one of 30, 45, or 60). We’ll archive a metric if it wasn’t used in the lookback period. The value here uses ISO 8061 duration format. Examples - 'P30D', 'P45D', 'P60D'",
		},
		"grace_period": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Grace period is an org level setting that applies to the newly created metrics. This allows customers to protect newly added metrics that users haven’t had the time to use in charts and detectors from being automatically archived The value here uses ISO 8061 duration format. Examples - 'P0D', 'P15D', 'P30D', 'P45D', 'P60D'",
		},
		"ruleset_limit": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "Org limit for the number of rulesets that can be created",
		},
	}
}

func decodeTerraform(data *schema.ResourceData) (*autoarch.AutomatedArchivalSettings, error) {
	settings := &autoarch.AutomatedArchivalSettings{
		Creator:        data.Get("creator").(*string),
		LastUpdatedBy:  data.Get("last_updated_by").(*string),
		Created:        data.Get("created").(*int64),
		LastUpdated:    data.Get("last_updated").(*int64),
		Version:        data.Get("version").(int64),
		Enabled:        data.Get("enabled").(bool),
		LookbackPeriod: data.Get("lookback_period").(string),
		GracePeriod:    data.Get("grace_period").(string),
		RulesetLimit:   data.Get("ruleset_limit").(*int32),
	}
	return settings, nil
}

func encodeTerraform(settings *autoarch.AutomatedArchivalSettings, data *schema.ResourceData) error {
	errs := multierr.Combine(
		data.Set("creator", settings.Creator),
		data.Set("last_updated_by", settings.LastUpdatedBy),
		data.Set("created", settings.Created),
		data.Set("last_updated", settings.LastUpdated),
		data.Set("version", settings.Version),
		data.Set("enabled", settings.Enabled),
		data.Set("lookback_period", settings.LookbackPeriod),
		data.Set("grace_period", settings.GracePeriod),
		data.Set("ruleset_limit", settings.RulesetLimit),
	)

	return errs
}
