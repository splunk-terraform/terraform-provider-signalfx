// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

var millistamps = map[rune]int{
	'm': 60 * 1000,               // Minute in Milliseconds
	'h': 60 * 60 * 1000,          // Hour in Milliseconds
	'd': 24 * 60 * 60 * 1000,     // Day in Milliseconds
	'w': 7 * 24 * 60 * 60 * 1000, // Week in Milliseconds
}

func isUnit(r rune) bool {
	_, ok := millistamps[r]
	return ok
}

// FromTimeRange converts a timerange value into a millisecond value.
// There is two supported syntax to the timerange:
// - No unit values: implies the value is defined in milliseconds
// - Unit values used: converts the sections into milliseconds based on unit
// These syntax styles can no be mixed
func FromTimeRangeToMilliseconds(tr string) (int, error) {
	if tr == "" {
		return 0, fmt.Errorf("invalid timerange: no value")
	}
	if !strings.HasPrefix(tr, "-") {
		return 0, fmt.Errorf("invalid timerange %q: no negative prefix", tr)
	}

	var (
		orig   = tr
		window = 0
		val    []rune
	)
	for _, r := range strings.TrimPrefix(tr, "-") {
		switch {
		case unicode.IsDigit(r):
			val = append(val, r)
		case isUnit(r):
			if len(val) == 0 {
				return 0, fmt.Errorf("invalid timerange %q: missing digits", orig)
			}
			v, err := strconv.Atoi(string(val))
			if err != nil {
				return 0, err
			}
			window += v * millistamps[r]
			val = val[:0]
		default:
			return 0, fmt.Errorf("invalid timerange %q: unknown value", orig)
		}
	}

	if len(val) != 0 {
		if window != 0 {
			return 0, fmt.Errorf("invalid timerange %q: mixed syntax used", orig)
		}
		// It is assumed that without a unit suffix,
		// the value is provided in milliseconds
		return strconv.Atoi(string(val))
	}

	return window, nil
}
