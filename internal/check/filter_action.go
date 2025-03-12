// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"fmt"
	"slices"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/integration"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func FilterAction() schema.SchemaValidateDiagFunc {
	return func(i any, p cty.Path) diag.Diagnostics {
		value, ok := i.(string)
		if !ok {
			return tfext.AsErrorDiagnostics(
				fmt.Errorf("expected %v to be of type string", i),
				p,
			)
		}

		actions := []string{
			string(integration.EXCLUDE),
			string(integration.INCLUDE),
		}

		if slices.Contains(actions, value) {
			return nil
		}

		return tfext.AsErrorDiagnostics(
			fmt.Errorf("value %q is not allowed; expected to be one of: %v", value, actions),
			p,
		)
	}
}
