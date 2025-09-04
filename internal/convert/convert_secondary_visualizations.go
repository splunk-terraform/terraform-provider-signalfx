// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"math"

	"github.com/signalfx/signalfx-go/chart"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/visual"
)

func ToChartSecondaryVisualization(in any) *chart.SecondaryVisualization {
	var (
		opt = in.(map[string]any)
		viz = &chart.SecondaryVisualization{}
	)

	cond := func(v float64) bool {
		// Ensure that the value is within the bounds of a 32 bit float.
		return math.Abs(v) < math.MaxFloat32
	}

	if v, ok := opt["gt"].(float64); ok {
		viz.Gt = common.AsPointerOnCondition(v, cond)
	}

	if v, ok := opt["gte"].(float64); ok {
		viz.Gte = common.AsPointerOnCondition(v, cond)
	}

	if v, ok := opt["lt"].(float64); ok {
		viz.Lt = common.AsPointerOnCondition(v, cond)
	}

	if v, ok := opt["lte"].(float64); ok {
		viz.Lte = common.AsPointerOnCondition(v, cond)
	}

	if c, ok := opt["color"].(string); ok {
		idx, ok := visual.NewColorScalePalette().ColorIndex(c)
		if ok {
			viz.PaletteIndex = common.AsPointer(idx)
		}
	}

	return viz
}
