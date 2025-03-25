// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import "slices"

// Unique processes various different lists of data
// and returns a unique list of all the values.
// The values are returned in order they are processed
// to provide a deterministic order.
func Unique[S ~[]E, E comparable](lists ...S) S {
	s := NewOrderedSet[E]()
	for _, l := range lists {
		s.Append(l...)
	}
	return slices.Collect(s.All())
}
