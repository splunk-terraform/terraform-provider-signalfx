// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"container/list"
	"iter"
)

// OrderedSet preserves the order of when elements
// where added into the set so that iterating through
// the values is deterministic since a default map does
// not ensure order when iterating through items, and
// can not assume that items should be sorted based on value.
type OrderedSet[E comparable] struct {
	seen  map[E]struct{}
	items *list.List
}

func NewOrderedSet[E comparable]() *OrderedSet[E] {
	return &OrderedSet[E]{
		seen:  make(map[E]struct{}),
		items: list.New(),
	}
}

func (s *OrderedSet[E]) All() iter.Seq[E] {
	return func(yield func(E) bool) {
		for node := s.items.Front(); node != nil; node = node.Next() {
			if !yield(node.Value.(E)) {
				return
			}
		}
	}
}

func (s *OrderedSet[E]) Add(e E) {
	if _, exist := s.seen[e]; !exist {
		s.seen[e] = struct{}{}
		s.items.PushBack(e)
	}
}

func (s *OrderedSet[E]) Append(items ...E) {
	for _, e := range items {
		s.Add(e)
	}
}
