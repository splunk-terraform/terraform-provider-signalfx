// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package organization

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestAcceptanceDataSourceMembers(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []resource.TestStep
	}{
		{
			name: "minimal",
			steps: []resource.TestStep{
				{
					Config: tftest.LoadConfig("testdata/data_members.tf"),
					Check: func(s *terraform.State) error {
						if !s.Empty() {
							return errors.New("expected no data returned")
						}
						return nil
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tftest.NewAcceptanceHandler(
				tftest.WithAcceptanceDataSources(map[string]*schema.Resource{
					DataSourceName: NewDataSource(),
				}),
			).
				Test(t, tc.steps)
		})
	}
}
