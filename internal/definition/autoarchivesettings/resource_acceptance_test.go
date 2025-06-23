package autoarchivesettings_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/autoarchivesettings"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestResourceAcceptance(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []resource.TestStep
	}{
		{
			name: "basic automated archival settings",
			steps: []resource.TestStep{
				{
					Config: tftest.LoadConfig("testdata/setting.tf"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_auto_archive_settings.setting", "creator", "Test Creator"),
						resource.TestCheckResourceAttr("signalfx_auto_archive_settings.setting", "last_updated_by", "Test Creator"),
						resource.TestCheckResourceAttr("signalfx_auto_archive_settings.setting", "created", "12340"),
						resource.TestCheckResourceAttr("signalfx_auto_archive_settings.setting", "last_updated", "12345"),
						resource.TestCheckResourceAttr("signalfx_auto_archive_settings.setting", "version", "1"),
						resource.TestCheckResourceAttr("signalfx_auto_archive_settings.setting", "enabled", "true"),
						resource.TestCheckResourceAttr("signalfx_auto_archive_settings.setting", "lookback_period", "P30D"),
						resource.TestCheckResourceAttr("signalfx_auto_archive_settings.setting", "grace_period", "P15D"),
						resource.TestCheckResourceAttr("signalfx_auto_archive_settings.setting", "ruleset_limit", "1000"),
					),
					ExpectNonEmptyPlan: false,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tftest.NewAcceptanceHandler(
				tftest.WithAcceptanceResources(map[string]*schema.Resource{
					autoarchivesettings.ResourceName: autoarchivesettings.NewResource(),
				}),
			).Test(t, tc.steps)
		})
	}
}
