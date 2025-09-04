// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"container/list"
	"iter"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/path"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func WalkResourceSchema(schema map[string]rschema.Attribute) iter.Seq2[path.Path, rschema.Attribute] {
	return func(yield func(path.Path, rschema.Attribute) bool) {
		queue := list.New()
		queue.PushBack(WalkedValue{p: path.Empty(), v: schema})
		for queue.Len() > 0 {
			var (
				elem = queue.Remove(queue.Front()).(WalkedValue)
				keys = slices.Sorted(maps.Keys(elem.v.(map[string]rschema.Attribute)))
			)
			for _, k := range keys {
				v := elem.v.(map[string]rschema.Attribute)[k]
				switch v := v.(type) {
				case rschema.MapNestedAttribute:
					queue.PushBack(WalkedValue{p: elem.Path().AtMapKey(k), v: v.NestedObject.Attributes})
				case rschema.SetNestedAttribute:
					queue.PushBack(WalkedValue{p: elem.Path().AtName(k), v: v.NestedObject.Attributes})
				case rschema.SingleNestedAttribute:
					queue.PushBack(WalkedValue{p: elem.Path().AtName(k), v: v.Attributes})
				case rschema.ListNestedAttribute:
					queue.PushBack(WalkedValue{p: elem.Path().AtName(k).AtListIndex(0), v: v.NestedObject.Attributes})
				}
				if !yield(elem.Path().AtName(k), v) {
					return
				}
			}
		}
	}
}
