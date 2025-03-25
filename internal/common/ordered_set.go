// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"iter"
	"slices"
)

// OrderedSet preserves the order of when elements
// where added into the set so that iterating through
// the values is deterministic since a default map does
// not ensure order when iterating through items, and
// can not assume that items should be sorted based on value.
type OrderedSet[E comparable] struct {
	seen  map[E]struct{}
	items []E
}

func NewOrderedSet[E comparable]() *OrderedSet[E] {
	return &OrderedSet[E]{
		seen: make(map[E]struct{}),
	}
}

func (s *OrderedSet[E]) All() iter.Seq[E] {
	return slices.Values(s.items)
}

func (s *OrderedSet[E]) Add(e E) {
	if _, exist := s.seen[e]; !exist {
		s.seen[e] = struct{}{}
		s.items = append(s.items, e)
	}
}

func (s *OrderedSet[E]) Append(items ...E) {
	for _, e := range items {
		s.Add(e)
	}
}
