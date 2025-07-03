// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package autoarchiveexemptmetric

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	automated_archival "github.com/signalfx/signalfx-go/automated-archival"
	"github.com/stretchr/testify/assert"
)

func TestSchemaDecode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		values map[string]any
		expect *[]automated_archival.ExemptMetric
		errVal string
	}{
		{
			name:   "no values provided",
			values: map[string]any{},
			expect: &[]automated_archival.ExemptMetric{},
			errVal: "",
		},
		{
			name: "all values provided",
			values: map[string]any{
				"exempt_metrics": []any{
					map[string]any{"name": "metric1"},
					map[string]any{"name": "metric2"},
				},
			},
			expect: &[]automated_archival.ExemptMetric{
				{Name: "metric1"},
				{Name: "metric2"},
			},
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data := schema.TestResourceDataRaw(t, newSchema(), tc.values)
			exemptMetrics, err := decodeTerraform(data)
			assert.Equal(t, tc.expect, exemptMetrics, "Must match expected exempt metrics")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.NoError(t, err, "Must not error when decoding exempt metrics")
			}
		})
	}
}

func TestSchemaEncode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		values *[]automated_archival.ExemptMetric
		errVal string
	}{
		{
			name:   "no values provided",
			values: &[]automated_archival.ExemptMetric{},
			errVal: "",
		},
		{
			name: "all values provided",
			values: &[]automated_archival.ExemptMetric{
				{Name: "metric1"},
				{Name: "metric2"},
			},
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data := schema.TestResourceDataRaw(t, newSchema(), map[string]any{})
			err := encodeTerraform(tc.values, data)
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error value")
			} else {
				assert.NoError(t, err, "Must not error when encoding exempt metrics")
			}
		})
	}
}
