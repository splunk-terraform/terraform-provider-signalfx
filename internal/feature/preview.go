// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package feature

import "sync/atomic"

// Preview allows for features to be guarded
// to allow users to opt in for the new functionality.
//
// A preview can be modified regardless of its global available state,
// however, a warning will be displayed on the console saying the
// preview will be removed in a feature release and no longer requires a user to opt in
// and if there is any issues, to contact support.
//
// By default, the preview is disabled by default and
// requires for the preview to marked as Global Available for it to default to true.
// Once marked as GA, this should be added into the change log for that release.
type Preview struct {
	_ struct{} // Enforce explicy key assignment

	enabled     *atomic.Bool
	available   bool
	description string
	introduced  string
}

func NewPreview(opts ...PreviewOption) (*Preview, error) {
	p := &Preview{
		enabled: new(atomic.Bool),
	}

	for _, opt := range opts {
		if err := opt.apply(p); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p Preview) Enabled() bool {
	return p.enabled.Load()
}

func (p Preview) SetEnabled(enabled bool) {
	p.enabled.Store(enabled)
}

func (p Preview) GlobalAvailable() bool {
	return p.available
}

func (p Preview) Description() string {
	return p.description
}

func (p Preview) Introduced() string {
	return p.introduced
}
