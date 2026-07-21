// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtimestamp

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatasourceTimestampSchema(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	DatasourceTimestamp{}.Schema(t.Context(), datasource.SchemaRequest{}, &resp)

	require.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.GetDescription())

	attributes := resp.Schema.GetAttributes()
	require.Len(t, attributes, 8)

	for _, name := range []string{"timezone", "year", "month", "day", "hour", "minute", "second", "value"} {
		attribute, ok := attributes[name]
		require.Truef(t, ok, "schema is missing the %q attribute", name)
		assert.NotEmptyf(t, attribute.GetDescription(), "%q must have a description", name)
	}

	year := attributes["year"]
	assert.True(t, year.IsRequired())
	assert.False(t, year.IsOptional())
	assert.False(t, year.IsComputed())

	for _, name := range []string{"timezone", "month", "day", "hour", "minute", "second"} {
		attribute := attributes[name]
		assert.Truef(t, attribute.IsOptional(), "%q must be optional", name)
		assert.Falsef(t, attribute.IsRequired(), "%q must not be required", name)
		assert.Falsef(t, attribute.IsComputed(), "%q must not be computed", name)
	}

	assert.Equal(t, types.StringType, attributes["timezone"].GetType())
	for _, name := range []string{"year", "month", "day", "hour", "minute", "second"} {
		assert.Equalf(t, types.Int32Type, attributes[name].GetType(), "%q must be an Int32 attribute", name)
	}

	value, ok := attributes["value"]
	require.True(t, ok, "schema is missing the value attribute")
	assert.True(t, value.IsComputed())
	assert.False(t, value.IsRequired())
	assert.False(t, value.IsOptional())
	assert.Equal(t, types.Int64Type, value.GetType())
}

func TestDatasourceTimestampRead(t *testing.T) {
	t.Parallel()

	newYork := time.FixedZone("EDT", -4*60*60)
	tests := []struct {
		name  string
		model DatasourceTimestampModel
		want  int64
	}{
		{
			name: "all values in UTC",
			model: DatasourceTimestampModel{
				Timezone: types.StringValue("UTC"),
				Year:     types.Int32Value(2024),
				Month:    types.Int32Value(int32(time.February)),
				Day:      types.Int32Value(29),
				Hour:     types.Int32Value(12),
				Minute:   types.Int32Value(34),
				Second:   types.Int32Value(56),
				Value:    types.Int64Null(),
			},
			want: time.Date(2024, time.February, 29, 12, 34, 56, 0, time.UTC).UnixMilli(),
		},
		{
			name: "IANA timezone",
			model: DatasourceTimestampModel{
				Timezone: types.StringValue("America/New_York"),
				Year:     types.Int32Value(2026),
				Month:    types.Int32Value(int32(time.July)),
				Day:      types.Int32Value(21),
				Hour:     types.Int32Value(14),
				Minute:   types.Int32Value(30),
				Second:   types.Int32Value(45),
				Value:    types.Int64Null(),
			},
			want: time.Date(2026, time.July, 21, 14, 30, 45, 0, newYork).UnixMilli(),
		},
		{
			name: "optional values omitted",
			model: DatasourceTimestampModel{
				Timezone: types.StringNull(),
				Year:     types.Int32Value(2026),
				Month:    types.Int32Null(),
				Day:      types.Int32Null(),
				Hour:     types.Int32Null(),
				Minute:   types.Int32Null(),
				Second:   types.Int32Null(),
				Value:    types.Int64Null(),
			},
			want: time.Date(2026, time.January, 0, 0, 0, 0, 0, time.UTC).UnixMilli(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ds := DatasourceTimestamp{}
			dsSchema := timestampDatasourceTestSchema(t, ds)
			config := timestampDatasourceTestConfig(t, dsSchema, tc.model)
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: dsSchema}}

			ds.Read(t.Context(), datasource.ReadRequest{Config: config}, &resp)

			require.Empty(t, resp.Diagnostics)
			var got DatasourceTimestampModel
			require.Empty(t, resp.State.Get(t.Context(), &got))

			want := tc.model
			want.Value = types.Int64Value(tc.want)
			assert.Equal(t, want, got)
		})
	}
}

func TestDatasourceTimestampReadInvalidTimezone(t *testing.T) {
	t.Parallel()

	ds := DatasourceTimestamp{}
	dsSchema := timestampDatasourceTestSchema(t, ds)
	model := DatasourceTimestampModel{
		Timezone: types.StringValue("Mars/Olympus_Mons"),
		Year:     types.Int32Value(2026),
		Month:    types.Int32Null(),
		Day:      types.Int32Null(),
		Hour:     types.Int32Null(),
		Minute:   types.Int32Null(),
		Second:   types.Int32Null(),
		Value:    types.Int64Null(),
	}
	config := timestampDatasourceTestConfig(t, dsSchema, model)
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dsSchema}}

	ds.Read(t.Context(), datasource.ReadRequest{Config: config}, &resp)

	require.Len(t, resp.Diagnostics, 1)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Equal(t, "Unable to parse timezone", resp.Diagnostics[0].Summary())
	assert.Contains(t, resp.Diagnostics[0].Detail(), "unknown time zone Mars/Olympus_Mons")
}

func timestampDatasourceTestSchema(t *testing.T, ds DatasourceTimestamp) schema.Schema {
	t.Helper()

	var resp datasource.SchemaResponse
	ds.Schema(t.Context(), datasource.SchemaRequest{}, &resp)
	require.Empty(t, resp.Diagnostics)

	return resp.Schema
}

func timestampDatasourceTestConfig(t *testing.T, dsSchema schema.Schema, model DatasourceTimestampModel) tfsdk.Config {
	t.Helper()

	state := tfsdk.State{Schema: dsSchema}
	require.Empty(t, state.Set(t.Context(), model))

	return tfsdk.Config{
		Raw:    state.Raw,
		Schema: dsSchema,
	}
}
