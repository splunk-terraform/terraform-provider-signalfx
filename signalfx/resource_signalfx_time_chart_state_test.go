// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeChartTimeRangeStateUpgradeV0(t *testing.T) {
	t.Parallel()
	assert.Contains(t, timeRangeV0().Schema, "time_range")
	state, err := timeRangeStateUpgradeV0(context.Background(), map[string]any{"time_range": "-10w2d"}, nil)
	require.NoError(t, err)
	assert.Equal(t, 6220800, state["time_range"])

	_, err = timeRangeStateUpgradeV0(context.Background(), map[string]any{"time_range": "not-a-range"}, nil)
	require.Error(t, err)
}
