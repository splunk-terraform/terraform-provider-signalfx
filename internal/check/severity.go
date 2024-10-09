// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"fmt"
	"slices"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func SeverityLevel() schema.SchemaValidateDiagFunc {
	return func(i interface{}, p cty.Path) diag.Diagnostics {
		value, ok := i.(string)
		if !ok {
			return tfext.AsErrorDiagnostics(
				fmt.Errorf("expected %v to be of type string", i),
				p,
			)
		}

		labels := []string{
			"Critical", "Major", "Minor", "Warning", "Info",
		}

		if slices.Contains(labels, value) {
			return nil
		}

		return tfext.AsErrorDiagnostics(
			fmt.Errorf("value %q is not allowed; must be one of: %v", value, labels),
			p,
		)
	}
}
