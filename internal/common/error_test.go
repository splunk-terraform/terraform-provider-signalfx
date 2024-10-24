// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go"
	"github.com/stretchr/testify/assert"
)

func TestOnError(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		err    error
		expect string
	}{
		{
			name:   "no error provided",
			err:    nil,
			expect: "id",
		},
		{
			name:   "not a response rror",
			err:    errors.New("derp"),
			expect: "id",
		},
		{
			name:   "uncatalogue response error",
			err:    &signalfx.ResponseError{},
			expect: "id",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := schema.TestResourceDataRaw(
				t,
				map[string]*schema.Schema{},
				map[string]any{},
			)
			data.SetId("id")

			assert.ErrorIs(t, tc.err, HandleError(context.Background(), tc.err, data), "Must return the same error")
			assert.Equal(t, tc.expect, data.Id(), "Must have the expected id")
		})
	}
}
