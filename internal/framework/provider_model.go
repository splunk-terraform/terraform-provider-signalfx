// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalframework

import "github.com/hashicorp/terraform-plugin-framework/types"

type ollyProviderModel struct {
	AuthToken           types.String `tfsdk:"auth_token"`
	APIURL              types.String `tfsdk:"api_url"`
	CustomAppURL        types.String `tfsdk:"custom_app_url"`
	TimeoutSeconds      types.Int64  `tfsdk:"timeout_seconds"`
	RetryMaxAttempts    types.Int32  `tfsdk:"retry_max_attempts"`
	RetryWaitMinSeconds types.Int64  `tfsdk:"retry_wait_min_seconds"`
	RetryWaitMaxSeconds types.Int64  `tfsdk:"retry_wait_max_seconds"`
	Email               types.String `tfsdk:"email"`
	Password            types.String `tfsdk:"password"`
	OrganizationID      types.String `tfsdk:"organization_id"`
	FeaturePreview      types.Map    `tfsdk:"feature_preview"`
	Tags                types.List   `tfsdk:"tags"`
	Teams               types.List   `tfsdk:"teams"`
}

func newDefaultOllyProviderModel() *ollyProviderModel {
	return &ollyProviderModel{
		AuthToken:           types.StringNull(),
		APIURL:              types.StringNull(),
		CustomAppURL:        types.StringNull(),
		TimeoutSeconds:      types.Int64Value(60),
		RetryMaxAttempts:    types.Int32Value(5),
		RetryWaitMinSeconds: types.Int64Value(1),
		RetryWaitMaxSeconds: types.Int64Value(10),
		Email:               types.StringNull(),
		Password:            types.StringNull(),
		OrganizationID:      types.StringNull(),
		FeaturePreview:      types.MapNull(types.BoolType),
		Tags:                types.ListNull(types.StringType),
		Teams:               types.ListNull(types.StringType),
	}
}
