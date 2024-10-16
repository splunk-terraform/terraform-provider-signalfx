// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func TimeRange() schema.SchemaValidateDiagFunc {
	return func(i any, p cty.Path) diag.Diagnostics {
		s, ok := i.(string)
		if !ok {
			return tfext.AsErrorDiagnostics(
				fmt.Errorf("expected %v to be type string", i),
				p,
			)
		}
		_, err := common.FromTimeRangeToMilliseconds(s)
		return tfext.AsErrorDiagnostics(err, p)
	}
}
