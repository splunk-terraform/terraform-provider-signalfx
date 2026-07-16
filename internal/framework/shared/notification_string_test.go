// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationStringListValidator(t *testing.T) {
	t.Parallel()
	implementation := NotificationStringListValidator()
	assert.Equal(t, implementation.Description(context.Background()), implementation.MarkdownDescription(context.Background()))

	for _, test := range []struct {
		name  string
		value types.List
		error bool
	}{
		{name: "null", value: types.ListNull(types.StringType)},
		{name: "unknown", value: types.ListUnknown(types.StringType)},
		{name: "valid", value: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("Email,alerts@example.com")})},
		{name: "invalid", value: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("Email,alerts@example.com"), types.StringValue("invalid"),
		}), error: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := &validator.ListResponse{}
			implementation.ValidateList(context.Background(), validator.ListRequest{
				Path: path.Root("notifications"), ConfigValue: test.value,
			}, response)
			assert.Equal(t, test.error, response.Diagnostics.HasError())
			if test.error {
				diagnostic, ok := response.Diagnostics[0].(diag.DiagnosticWithPath)
				require.True(t, ok)
				assert.Equal(t, `notifications[1]`, diagnostic.Path().String())
			}
		})
	}
}

func TestNotificationStringConversions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	apiValues, diagnostics := NotificationStringsToAPI(ctx, types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("Email,alerts@example.com"), types.StringValue("Slack,credential-id,alerts"),
	}))
	require.False(t, diagnostics.HasError())
	require.Len(t, apiValues, 2)
	assert.Equal(t, "Email", apiValues[0].Type)
	assert.Equal(t, "Slack", apiValues[1].Type)

	value, diagnostics := NotificationStringsFromAPI(ctx, types.ListNull(types.StringType), apiValues)
	require.False(t, diagnostics.HasError())
	assert.Equal(t, []attr.Value{
		types.StringValue("Email,alerts@example.com"), types.StringValue("Slack,credential-id,alerts"),
	}, value.Elements())

	nullValue, diagnostics := NotificationStringsFromAPI(ctx, types.ListNull(types.StringType), nil)
	assert.False(t, diagnostics.HasError())
	assert.True(t, nullValue.IsNull())

	emptyValue, diagnostics := NotificationStringsFromAPI(ctx, types.ListValueMust(types.StringType, nil), nil)
	assert.False(t, diagnostics.HasError())
	assert.Empty(t, emptyValue.Elements())
}

func TestNotificationStringConversionErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	_, diagnostics := NotificationStringsToAPI(ctx, types.ListUnknown(types.StringType))
	assert.True(t, diagnostics.HasError())

	_, diagnostics = NotificationStringsToAPI(ctx, types.ListValueMust(types.StringType, []attr.Value{types.StringValue("invalid")}))
	assert.True(t, diagnostics.HasError())

	_, diagnostics = NotificationStringsFromAPI(ctx, types.ListNull(types.StringType), []*notification.Notification{{
		Type: "Unknown", Value: struct{}{},
	}})
	assert.True(t, diagnostics.HasError())
}
