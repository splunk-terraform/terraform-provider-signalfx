// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package detector

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestStateV0(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		map[string]*schema.Schema{
			"time_range": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		stateV0State().Schema,
		"Must match the expected value",
	)
}

func TestStateMigrationV0(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		state  map[string]any
		expect map[string]any
		errVal string
	}{
		{
			name:   "empty state",
			state:  map[string]any{},
			expect: map[string]any{},
			errVal: "",
		},
		{
			name: "invalid time range set",
			state: map[string]any{
				"time_range": "friday",
			},
			expect: nil,
			errVal: "invalid timerange \"friday\": no negative prefix",
		},
		{
			name: "valid timerange set",
			state: map[string]any{
				"time_range": "-10w2d",
			},
			expect: map[string]any{
				"time_range": 6220800,
			},
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := stateMigrationV0(context.Background(), tc.state, nil)

			assert.Equal(t, tc.expect, actual, "Must match the expected state")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error")
			} else {
				assert.NoError(t, err, "Must not report an error")
			}
		})
	}
}
