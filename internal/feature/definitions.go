// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package feature

// This file is used to define all the existing feature previews
// that are currently used within the provider.
// Having the values defined in one place makes it easier to
// update and maintain these values.
// Furthermore, if these are to be used within an autogen script
// then it does not require loading each individual module
// which will speed up documentation generation.
//
// When defining a new feature preview, it should:
// - named in terms of scope and and feature
// 	- ie: `provider.<feature>`, `detectors.<feature>`
// - Add a description that informs the user of what will happen once enabled.
// - Set the version added in (this helps sorting oldest previews to newest)

const (
	PreviewProviderTeams = "provider.teams"
)

var (
	_ = GetGlobalRegistry().MustRegister(
		PreviewProviderTeams,
		WithPreviewDescription("Allows for team(s) to set at a provider level, and apply to all applicable resources"),
		WithPreviewAddInVersion("v9.9.1"),
	)
)
