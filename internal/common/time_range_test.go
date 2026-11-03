// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromTimeRange(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		window string
		expect int
		errVal string
	}{
		{
			name:   "no value provided",
			window: "",
			expect: 0,
			errVal: "invalid timerange: no value",
		},
		{
			name:   "no negative range provided",
			window: "10h",
			expect: 0,
			errVal: "invalid timerange \"10h\": no negative prefix",
		},
		{
			name:   "mixed syntax",
			window: "-10h100",
			expect: 0,
			errVal: "invalid timerange \"-10h100\": mixed syntax used",
		},
		{
			name:   "invalid string",
			window: "-10l",
			expect: 0,
			errVal: "invalid timerange \"-10l\": unknown value",
		},
		{
			name:   "no digits",
			window: "-h",
			expect: 0,
			errVal: "invalid timerange \"-h\": missing digits",
		},
		{
			name:   "simple timerange",
			window: "-1w",
			expect: 604800000,
			errVal: "",
		},
		{
			name:   "multiple units",
			window: "-4w10d",
			expect: 3283200000,
			errVal: "",
		},
		{
			name:   "compound hours and minutes",
			window: "-1h30m",
			expect: 5400000,
		},
		{
			name:   "seconds used",
			window: "-30s",
			expect: 30000,
		},
		{
			name:   "seconds with other units",
			window: "-1m30s",
			expect: 90000,
		},
		{
			name:   "milliseconds used",
			window: "-10",
			expect: 10,
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := FromTimeRangeToMilliseconds(tc.window)
			assert.Equal(t, tc.expect, actual, "Must match the expected value")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error message")
			} else {
				assert.NoError(t, err, "Must not error")
			}
		})
	}
}

func TestFromTimeRangeOverflow(t *testing.T) {
	maxMillis := strconv.Itoa(math.MaxInt)
	actual, err := FromTimeRangeToMilliseconds("-" + maxMillis)
	require.NoError(t, err)
	assert.Equal(t, math.MaxInt, actual)

	maxSeconds := math.MaxInt / 1000
	actual, err = FromTimeRangeToMilliseconds("-" + strconv.Itoa(maxSeconds) + "s")
	require.NoError(t, err)
	assert.Equal(t, maxSeconds*1000, actual)

	for _, value := range []string{
		"-" + strconv.FormatUint(uint64(math.MaxInt)+1, 10),
		"-" + strconv.Itoa(maxSeconds+1) + "s",
		"-" + strconv.Itoa(maxSeconds) + "s1s",
	} {
		t.Run(value, func(t *testing.T) {
			_, err := FromTimeRangeToMilliseconds(value)
			assert.ErrorContains(t, err, "overflows milliseconds")
		})
	}
}
