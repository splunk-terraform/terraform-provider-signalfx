// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package detector

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func stateV0State() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"time_range": {Type: schema.TypeString, Optional: true},
		},
	}
}

func stateMigrationV0(ctx context.Context, state map[string]any, _ any) (map[string]any, error) {
	tflog.Debug(ctx, "Upgrading detector state", tfext.NewLogFields().JSON("state", state))

	if tr, ok := state["time_range"].(string); ok {
		millis, err := common.FromTimeRangeToMilliseconds(tr)
		if err != nil {
			return nil, err
		}
		// Convert from millis back to seconds
		state["time_range"] = millis / 1000
	}

	return state, nil
}
