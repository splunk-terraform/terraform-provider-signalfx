// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package autoarchiveexemptmetric

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	automated_archival "github.com/signalfx/signalfx-go/automated-archival"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
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
			name: "all required values provided",
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
		{
			name: "invalid type for exempt_metrics",
			values: map[string]any{
				"exempt_metrics": "not a list",
			},
			expect: &[]automated_archival.ExemptMetric{},
			errVal: "",
		},
		{
			name: "empty exempt_metrics list",
			values: map[string]any{
				"exempt_metrics": []any{},
			},
			expect: &[]automated_archival.ExemptMetric{},
			errVal: "",
		},
		{
			name: "missing required field",
			values: map[string]any{
				"exempt_metrics": []any{
					map[string]any{"name": "metric1"},
					map[string]any{}, // Missing 'name'
				},
			},
			expect: &[]automated_archival.ExemptMetric{
				{Name: "metric1"},
				{Name: ""},
			},
			errVal: "",
		},
		{
			name: "exempt metric with additional fields",
			values: map[string]any{
				"exempt_metrics": []any{
					map[string]any{
						"name":            "metric1",
						"creator":         "user1",
						"last_updated_by": "user2",
						"created":         1622547800,
						"last_updated":    1622547800,
					},
				},
			},
			expect: &[]automated_archival.ExemptMetric{
				{
					Name:          "metric1",
					Creator:       common.AsPointer("user1"),
					LastUpdatedBy: common.AsPointer("user2"),
					Created:       common.AsPointer[int64](1622547800),
					LastUpdated:   common.AsPointer[int64](1622547800),
				},
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
		{
			name: "single exempt metric with additional fields",
			values: &[]automated_archival.ExemptMetric{
				{
					Name:          "metric1",
					Creator:       common.AsPointer("user1"),
					LastUpdatedBy: common.AsPointer("user2"),
					Created:       common.AsPointer[int64](1622547800),
					LastUpdated:   common.AsPointer[int64](1622547800),
				},
			},
			errVal: "",
		},
		{
			name:   "nil values",
			values: nil,
			errVal: "",
		},
		{
			name:   "empty values",
			values: &[]automated_archival.ExemptMetric{},
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
