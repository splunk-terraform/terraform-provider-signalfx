// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
)

// timeRangeV0 is retained for the excluded SDKv2 time chart state upgrader.
func timeRangeV0() *schema.Resource {
	return &schema.Resource{Schema: map[string]*schema.Schema{
		"time_range": {Type: schema.TypeString, Optional: true},
	}}
}

func timeRangeStateUpgradeV0(_ context.Context, rawState map[string]any, _ any) (map[string]any, error) {
	log.Printf("[DEBUG] SignalFx: Upgrading time chart state %v", rawState["time_range"])
	if timeRange, ok := rawState["time_range"].(string); ok {
		milliseconds, err := common.FromTimeRangeToMilliseconds(timeRange)
		if err != nil {
			return rawState, err
		}
		rawState["time_range"] = milliseconds / 1000
	}
	return rawState, nil
}
