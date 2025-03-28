// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"fmt"
	"net/mail"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func Email(i any, p cty.Path) diag.Diagnostics {
	v, ok := i.(string)
	if !ok {
		return tfext.AsErrorDiagnostics(
			fmt.Errorf("expected %v to be of type string", i),
			p,
		)
	}
	_, err := mail.ParseAddress(v)
	return tfext.AsErrorDiagnostics(err, p)
}
