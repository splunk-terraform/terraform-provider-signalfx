// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestSchemaListAll(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		schema *schema.Set
		F      Func[any, any]
		expect []any
	}{
		{
			name:   "nil set",
			schema: nil,
			F:      func(s any) any { return s },
			expect: nil,
		},
		{
			name:   "no values set",
			schema: schema.NewSet(schema.HashInt, nil),
			F:      func(s any) any { return s },
			expect: []any{},
		},
		{
			name:   "int set",
			schema: schema.NewSet(schema.HashInt, []any{1, 2}),
			F:      func(s any) any { return s },
			expect: []any{1, 2},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expect, SchemaListAll(tc.schema, tc.F))
		})
	}
}
