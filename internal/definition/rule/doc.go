// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package rule does not export a resource or data source
// since it doesn't have its own dedicated API endpoint.
// However, the rule schema definition is shared across
// many resources and data sources so it is abstracted
// to its own package to avoid circular imports.
package rule
