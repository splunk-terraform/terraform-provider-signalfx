// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package feature

import (
	"context"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewRegistry(), "Must have a valid type when called")
}

func TestRegistryAll(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		entries map[string][]PreviewOption
	}{
		{
			name:    "no values",
			entries: map[string][]PreviewOption{},
		},
		{
			name: "one feature",
			entries: map[string][]PreviewOption{
				"chart_validation": {},
			},
		},
		{
			name: "multiple entries",
			entries: map[string][]PreviewOption{
				"chart_validation": {},
				"new_ui_elements":  {},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := NewRegistry()
			for feature, opts := range tc.entries {
				p, err := r.Register(feature, opts...)
				assert.NoError(t, err, "Must not error creating preview")
				assert.NotNil(t, p, "Must have a valid preview value")
			}

			features := maps.Collect(r.All())
			assert.Len(t, features, len(tc.entries), "Must match the expected length of entries")
			for feature := range tc.entries {
				assert.Contains(t, features, feature, "Must contain the expected feature")
			}
		})
	}
}

func TestRegistryRegister(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		feature string
		opts    []PreviewOption
		errVal  string
	}{
		{
			name:    "empty feature name",
			feature: "",
			opts:    []PreviewOption{},
			errVal:  "feature \"\" does not match expected naming format",
		},
		{
			name:    "use of hyphen",
			feature: "my-feature-01",
			opts:    []PreviewOption{},
			errVal:  "",
		},
		{
			name:    "use of dot notation",
			feature: "my.feature-01",
			opts:    []PreviewOption{},
			errVal:  "",
		},
		{
			name:    "bad preview options",
			feature: "my.feature",
			opts: []PreviewOption{
				WithPreviewDescription(""),
			},
			errVal: "adding empty description",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := NewRegistry().Register(tc.feature, tc.opts...)
			if tc.errVal != "" {
				assert.Nil(t, p, "Must have a returned a nil value")
				assert.EqualError(t, err, tc.errVal, "Must match the expected error value")
			} else {
				assert.NotNil(t, p, "Must have a valid preview")
				assert.NoError(t, err, "Must not returned an error")
			}
		})
	}
}

func TestRegistryMustRegister(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		_ = NewRegistry().MustRegister("my_feature_1")
	})

	assert.Panics(t, func() {
		_ = NewRegistry().MustRegister("")
	})

	assert.Panics(t, func() {
		r := NewRegistry()
		_ = r.MustRegister("my-feature-01")
		_ = r.MustRegister("my-feature-01")
	})
}

func TestRegistryConfigure(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		opts    []PreviewOption
		feature string
		errVal  string
	}{
		{
			name:    "default preview",
			opts:    []PreviewOption{},
			feature: "feature-01",
			errVal:  "",
		},
		{
			name:    "unset preview",
			opts:    []PreviewOption{},
			feature: "feature-02",
			errVal:  "no preview with id \"feature-02\" found",
		},
		{
			name: "globally available",
			opts: []PreviewOption{
				WithPreviewEnabled(),
				WithPreviewGlobalAvailable(),
			},
			feature: "feature-01",
			errVal:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reg := NewRegistry()
			_, err := reg.Register("feature-01", tc.opts...)
			require.NoError(t, err, "Must not error when register a feature preview")

			err = reg.Configure(context.Background(), tc.feature, true)
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.NoError(t, err, "Must not return an error")
			}
		})
	}
}
