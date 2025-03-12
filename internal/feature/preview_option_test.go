// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package feature

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreviewOption(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		fn     PreviewOption
		errVal string
	}{
		{
			name:   "unset preview option",
			fn:     nil,
			errVal: "function is nil",
		},
		{
			name:   "Global Available",
			fn:     WithPreviewGlobalAvailable(),
			errVal: "",
		},
		{
			name:   "Bad AddedInVersion",
			fn:     WithPreviewAddInVersion("v2"),
			errVal: "version string \"v2\" needs to be in format vX.Y[.+]",
		},
		{
			name:   "Valid AddedInVerison",
			fn:     WithPreviewAddInVersion("v2.1.0"),
			errVal: "",
		},
		{
			name:   "Invalid Description",
			fn:     WithPreviewDescription(""),
			errVal: "adding empty description",
		},
		{
			name:   "Valid Description",
			fn:     WithPreviewDescription("Allows for new fancy thing"),
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.fn.apply(&Preview{enabled: new(atomic.Bool)})
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.NoError(t, err, "Must not error")
			}
		})
	}
}
