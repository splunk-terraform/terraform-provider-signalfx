// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/visual"
)

func ColorName() schema.SchemaValidateDiagFunc {
	return func(i any, p cty.Path) diag.Diagnostics {
		s, ok := i.(string)
		if !ok {
			return tfext.AsErrorDiagnostics(
				fmt.Errorf("expected %v to be of type string", i),
				p,
			)
		}
		cp := visual.NewColorPalette()
		if _, exist := cp.ColorIndex(s); exist {
			return nil
		}
		return tfext.AsErrorDiagnostics(
			fmt.Errorf("value %q is not allowed; must be one of %v", s, cp.Names()),
			p,
		)
	}
}

func ColorHexValue() schema.SchemaValidateDiagFunc {
	return func(i any, p cty.Path) diag.Diagnostics {
		s, ok := i.(string)
		if !ok {
			return tfext.AsErrorDiagnostics(
				fmt.Errorf("expected %v to be of type string", i),
				p,
			)
		}
		if matched, _ := regexp.MatchString(`^#[A-Fa-f0-9]{6}$`, s); !matched {
			return tfext.AsErrorDiagnostics(
				fmt.Errorf("value %q is not a valid hex color code", s),
				p,
			)
		}
		return nil
	}
}
