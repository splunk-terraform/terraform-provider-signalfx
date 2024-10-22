// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
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
		total   = 0
		partial = 0
	)

	for _, r := range strings.TrimPrefix(tr, "-") {
		switch {
		case unicode.IsDigit(r):
			partial = partial*10 + int(r-'0')
		case isUnit(r):
			if partial == 0 {
				return 0, fmt.Errorf("invalid timerange %q: missing digits", tr)
			}
			total, partial = (total + partial*millistamps[r]), 0
		default:
			return 0, fmt.Errorf("invalid timerange %q: unknown value", tr)
		}
	}

	if partial != 0 && total != 0 {
		return 0, fmt.Errorf("invalid timerange %q: mixed syntax used", tr)
	}

	return total + partial, nil
}
