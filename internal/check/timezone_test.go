package check

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/assert"
)

func TestTimeZoneLocation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		val    any
		expect diag.Diagnostics
	}{
		{
			name: "no value provided",
			val:  nil,
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected <nil> as string"},
			},
		},
		{
			name:   "default value provided",
			val:    "UTC",
			expect: nil,
		},
		{
			name:   "configured tz location set",
			val:    "Australia/Adelaide",
			expect: nil,
		},
		{
			name: "Invalid tz location set",
			val:  "planet/earth",
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "unknown time zone planet/earth"},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := TimeZoneLocation()(tc.val, cty.Path{})
			assert.Equal(t, tc.expect, d, "Must match the expected value")
		})
	}
}
