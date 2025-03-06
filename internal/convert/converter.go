// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// SliceAll preallocates the require heap memory for the slice,
// then applies the converter function on each element,
// which is stored in the same positional value in the slice.
func SliceAll[S ~[]In, In any, Out any](s S, converter Func[In, Out]) []Out {
	out := make([]Out, len(s))
	for i, v := range s {
		out[i] = converter(v)
	}
	return out
}

// SchemaListAll will attempt to cast in as a [*schema.Set]
// and if it can not convert or if the set size is zero, nil is returned.
// Otherwise, the converter is applied to each item in the set and returned as a slice.
func SchemaListAll[Out any](in any, converter Func[any, Out]) []Out {
	set, ok := in.(*schema.Set)
	if !ok || set == nil {
		return nil
	}
	return SliceAll(set.List(), converter)
}
