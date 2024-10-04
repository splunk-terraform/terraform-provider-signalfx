package check

import (
	"fmt"
	"slices"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func ValueUnit() schema.SchemaValidateDiagFunc {
	return func(i interface{}, p cty.Path) diag.Diagnostics {
		s, ok := i.(string)
		if !ok {
			return tfext.AsErrorDiagnostics(
				fmt.Errorf("expected %v to be type string", i),
				p,
			)
		}

		units := []string{
			"Bit",
			"Kilobit",
			"Megabit",
			"Gigabit",
			"Terabit",
			"Petabit",
			"Exabit",
			"Zettabit",
			"Yottabit",
			"Byte",
			"Kibibyte",
			"Mebibyte",
			"Gibibyte",
			"Tebibyte",
			"Pebibyte",
			"Exbibyte",
			"Zebibyte",
			"Yobibyte",
			"Nanosecond",
			"Microsecond",
			"Millisecond",
			"Second",
			"Minute",
			"Hour",
			"Day",
			"Week",
		}

		if slices.Contains(units, s) {
			return nil
		}
		return tfext.AsErrorDiagnostics(
			fmt.Errorf("expected %q to be one of %v", s, units),
			p,
		)
	}
}
