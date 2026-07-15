// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"container/list"
	"iter"
	"maps"
	"slices"

	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

func WalkDataSourceSchema(attributes map[string]dschema.Attribute) iter.Seq2[path.Path, dschema.Attribute] {
	return func(yield func(path.Path, dschema.Attribute) bool) {
		queue := list.New()
		queue.PushBack(WalkedValue{p: path.Empty(), v: attributes})
		for queue.Len() > 0 {
			elem := queue.Remove(queue.Front()).(WalkedValue)
			values := elem.v.(map[string]dschema.Attribute)
			for _, key := range slices.Sorted(maps.Keys(values)) {
				attribute := values[key]
				switch nested := attribute.(type) {
				case dschema.MapNestedAttribute:
					queue.PushBack(WalkedValue{p: elem.Path().AtMapKey(key), v: nested.NestedObject.Attributes})
				case dschema.SetNestedAttribute:
					queue.PushBack(WalkedValue{p: elem.Path().AtName(key), v: nested.NestedObject.Attributes})
				case dschema.SingleNestedAttribute:
					queue.PushBack(WalkedValue{p: elem.Path().AtName(key), v: nested.Attributes})
				case dschema.ListNestedAttribute:
					queue.PushBack(WalkedValue{p: elem.Path().AtName(key).AtListIndex(0), v: nested.NestedObject.Attributes})
				}
				if !yield(elem.Path().AtName(key), attribute) {
					return
				}
			}
		}
	}
}
