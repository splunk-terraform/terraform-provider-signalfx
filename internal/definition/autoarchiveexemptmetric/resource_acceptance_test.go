package autoarchiveexemptmetric_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/autoarchiveexemptmetric"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestResourceAcceptance(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []resource.TestStep
	}{
		{
			name: "basic automated archival exempt metric",
			steps: []resource.TestStep{
				{
					Config: tftest.LoadConfig("testdata/resource_exempt_metric.tf"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_auto_archive_exempt_metric.exempt_metrics", "name", "exempt_metric_1"),
					),
					ExpectNonEmptyPlan: false,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tftest.NewAcceptanceHandler(
				tftest.WithAcceptanceResources(map[string]*schema.Resource{
					autoarchiveexemptmetric.ResourceName: autoarchiveexemptmetric.NewResource(),
				}),
			).Test(t, tc.steps)
		})
	}
}
