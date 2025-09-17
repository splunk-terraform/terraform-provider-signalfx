// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalframework

import (
	"os"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type OllyProviderModel struct {
	APIURL              types.String `tfsdk:"api_url"`
	AuthToken           types.String `tfsdk:"auth_token"`
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

func newDefaultOllyProviderModel() *OllyProviderModel {
	return &OllyProviderModel{
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

// EnsureDefaults sets the default values for the provider model fields if they are not already set.
// This is due to the fact that the Terraform Framework does not support default values for provider schema attributes.
func (model *OllyProviderModel) EnsureDefaults() {
	if data, ok := os.LookupEnv("SFX_AUTH_TOKEN"); ok && model.AuthToken.IsNull() {
		model.AuthToken = types.StringValue(data)
	}
	if data, ok := os.LookupEnv("SFX_API_URL"); ok && model.APIURL.IsNull() {
		model.APIURL = types.StringValue(data)
	}
	if model.TimeoutSeconds.IsNull() {
		model.TimeoutSeconds = types.Int64Value(60)
	}
	if model.RetryMaxAttempts.IsNull() {
		model.RetryMaxAttempts = types.Int32Value(5)
	}
	if model.RetryWaitMinSeconds.IsNull() {
		model.RetryWaitMinSeconds = types.Int64Value(1)
	}
	if model.RetryWaitMaxSeconds.IsNull() {
		model.RetryWaitMaxSeconds = types.Int64Value(10)
	}
}
