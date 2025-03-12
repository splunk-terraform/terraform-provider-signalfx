// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package feature

import "sync"

var (
	once   sync.Once
	global *Registry
)

// GetGlobalRegistry is used to allow for components to guard
// functionality with a preview.
// Any component that depends on using registry should allow
// passing a registry value to make it possible to test functionality
// without triggering the race detector.
func GetGlobalRegistry() *Registry {
	if global == nil {
		once.Do(func() {
			global = NewRegistry()
		})
	}
	return global
}
