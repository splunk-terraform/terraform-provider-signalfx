// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"math"
	"testing"

	"github.com/signalfx/signalfx-go/chart"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
)

func TestToChartSecondaryVisualization(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		input  any
		expect *chart.SecondaryVisualization
	}{
		{
			name:   "empty",
			input:  map[string]any{},
			expect: &chart.SecondaryVisualization{},
		},
		{
			name: "set color value",
			input: map[string]any{
				"color": "lilac",
			},
			expect: &chart.SecondaryVisualization{
				PaletteIndex: common.AsPointer[int32](12),
			},
		},
		{
			name: "setting gt value",
			input: map[string]any{
				"gt": 3.14,
			},
			expect: &chart.SecondaryVisualization{
				Gt: common.AsPointer(3.14),
			},
		},
		{
			name: "setting gte",
			input: map[string]any{
				"gte": 13.37,
			},
			expect: &chart.SecondaryVisualization{
				Gte: common.AsPointer(13.37),
			},
		},
		{
			name: "setting lt",
			input: map[string]any{
				"lt": 1.089,
			},
			expect: &chart.SecondaryVisualization{
				Lt: common.AsPointer(1.089),
			},
		},
		{
			name: "setting lte",
			input: map[string]any{
				"lte": 47.09,
			},
			expect: &chart.SecondaryVisualization{
				Lte: common.AsPointer(47.09),
			},
		},
		{
			name: "each value exceeds bounds",
			input: map[string]any{
				"lt":  math.MaxFloat32 + 4,
				"lte": math.MaxFloat32,
				"gt":  -math.MaxFloat32,
				"gte": -math.MaxFloat32 - 4,
			},
			expect: &chart.SecondaryVisualization{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expect, ToChartSecondaryVisualization(tc.input))
		})
	}
}
