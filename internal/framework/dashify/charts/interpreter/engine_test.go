// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package interpreter

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testProperty(kind propertyType, path ...string) property {
	return property{Type: kind, Optional: true, Description: "Test field.", Mapping: mapping{Path: path}}
}
func testEngine(t *testing.T, properties map[string]property) *Engine {
	t.Helper()
	e, err := newEngine([]definition{{Type: "test", Version: 1, Element: "o11y:Test", Properties: properties}})
	require.NoError(t, err)
	return e
}
func testValue(t *testing.T, e *Engine, overrides map[string]attr.Value) types.Object {
	t.Helper()
	attributes, err := e.Attributes("test")
	require.NoError(t, err)
	attrTypes := attributeTypes(attributes)
	values := map[string]attr.Value{}
	for name, typ := range attrTypes {
		value, err := typ.ValueFromTerraform(t.Context(), tftypes.NewValue(typ.TerraformType(t.Context()), nil))
		require.NoError(t, err)
		values[name] = value
	}
	for name, value := range overrides {
		values[name] = value
	}
	value, d := types.ObjectValue(attrTypes, values)
	require.False(t, d.HasError(), d)
	return value
}

func TestValidationDefersUnknownAndEnforcesConditionalCollections(t *testing.T) {
	mode := testProperty("enum", "chart", "mode")
	mode.Values = []string{"Scale", "None"}
	scale := testProperty("list", "chart", "scale")
	scale.Item = &item{Type: "string"}
	scale.RequiredWhen = &condition{Field: "mode", Value: "Scale"}
	scale.RequiresValue = &condition{Field: "mode", Value: "Scale"}
	e := testEngine(t, map[string]property{"mode": mode, "scale": scale})
	unknown := testValue(t, e, map[string]attr.Value{"mode": types.StringValue("Scale"), "scale": types.ListUnknown(types.StringType)})
	d := e.Validate("test", unknown)
	assert.False(t, d.HasError(), "unknown rejected during plan: %v", d)
	empty := testValue(t, e, map[string]attr.Value{"mode": types.StringValue("Scale"), "scale": types.ListValueMust(types.StringType, []attr.Value{})})
	d = e.Validate("test", empty)
	assert.True(t, d.HasError(), "conditional empty collection accepted")
	wrong := testValue(t, e, map[string]attr.Value{"mode": types.StringValue("None"), "scale": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("red")})})
	d = e.Validate("test", wrong)
	assert.True(t, d.HasError(), "required sibling value ignored")
}

func TestUnicodeLengthAndFiniteFloats(t *testing.T) {
	minimum := 2
	title := testProperty("string", "widget", "title")
	title.MinLength = &minimum
	e := testEngine(t, map[string]property{"title": title, "limit": testProperty("float", "chart", "limit")})
	for _, limit := range []float64{math.Inf(1), math.Inf(-1)} {
		value := testValue(t, e, map[string]attr.Value{"title": types.StringValue("你好"), "limit": types.Float64Value(limit)})
		_, d := e.Encode(context.Background(), "test", value)
		assert.True(t, d.HasError(), "nonfinite write accepted: %v", limit)
	}
	value := testValue(t, e, map[string]attr.Value{"title": types.StringValue("你")})
	d := e.Validate("test", value)
	assert.True(t, d.HasError(), "UTF-8 bytes counted as characters")
}

func TestNestedCollectionsRoundTrip(t *testing.T) {
	label := testProperty("string", "label")
	list := testProperty("list", "chart", "windows")
	list.Item = &item{Properties: map[string]property{"label": label}}
	items := testProperty("map", "chart", "namedWindows")
	items.Item = list.Item
	e := testEngine(t, map[string]property{"windows": list, "named_windows": items})
	attributes, err := e.Attributes("test")
	require.NoError(t, err)
	objectType := attributes["windows"].GetType().(types.ListType).ElemType.(types.ObjectType)
	window := types.ObjectValueMust(objectType.AttrTypes, map[string]attr.Value{"label": types.StringValue("sample")})
	configured := testValue(t, e, map[string]attr.Value{"windows": types.ListValueMust(objectType, []attr.Value{window}), "named_windows": types.MapValueMust(objectType, map[string]attr.Value{"a": window})})
	spec, d := e.Encode(context.Background(), "test", configured)
	require.False(t, d.HasError(), d)
	decoded, leftovers, d := e.Decode(context.Background(), "test", jsonSpec(t, spec), types.ObjectNull(nil))
	require.False(t, d.HasError(), d)
	assert.Empty(t, leftovers)
	assert.True(t, decoded.Equal(configured), "nested collections changed: expected %s, got %s", configured, decoded)
	empty := testValue(t, e, map[string]attr.Value{"windows": types.ListValueMust(objectType, []attr.Value{}), "named_windows": types.MapValueMust(objectType, map[string]attr.Value{})})
	spec, d = e.Encode(context.Background(), "test", empty)
	require.False(t, d.HasError(), d)
	decoded, leftovers, d = e.Decode(context.Background(), "test", jsonSpec(t, spec), empty)
	require.False(t, d.HasError(), d)
	assert.Empty(t, leftovers)
	assert.True(t, decoded.Equal(empty), "empty collections changed: expected %s, got %s", empty, decoded)
}

func TestTableGroupingDecodesCurrentArrays(t *testing.T) {
	e, err := Default()
	require.NoError(t, err)
	for _, test := range []struct {
		name     string
		wire     any
		want     []attr.Value
		wantWire []any
	}{
		{name: "no grouping", wire: []any{}, want: []attr.Value{}, wantWire: []any{}},
		{name: "one grouping", wire: []any{"host"}, want: []attr.Value{types.StringValue("host")}, wantWire: []any{"host"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := map[string]any{
				"<o11y:TableChart>": []any{},
				"widget":            map[string]any{},
				"datasource":        map[string]any{"program": "A"},
				"chart":             map[string]any{"groupBy": test.wire},
			}
			value, leftovers, diagnostics := e.Decode(t.Context(), "metrics_table", spec, types.ObjectNull(nil))
			require.False(t, diagnostics.HasError(), diagnostics)
			assert.Empty(t, leftovers)
			grouping := value.Attributes()["group_by"].(types.List)
			assert.False(t, grouping.IsNull(), "grouping must preserve an empty list rather than null")
			assert.True(t, grouping.Equal(types.ListValueMust(types.StringType, test.want)), "unexpected grouping: expected %v, got %s", test.want, grouping)
			encoded, diagnostics := e.Encode(t.Context(), "metrics_table", value)
			require.False(t, diagnostics.HasError(), diagnostics)
			assert.Equal(t, test.wantWire, encoded["chart"].(map[string]any)["groupBy"].([]any), "encoded grouping changed")
		})
	}
	tooMany := map[string]any{
		"<o11y:TableChart>": []any{},
		"widget":            map[string]any{},
		"datasource":        map[string]any{"program": "A"},
		"chart":             map[string]any{"groupBy": []any{"host", "region"}},
	}
	value, _, diagnostics := e.Decode(t.Context(), "metrics_table", jsonSpec(t, tooMany), types.ObjectNull(nil))
	require.Empty(t, diagnostics, "stored grouping outside the configured limit should decode without diagnostics")
	_, diagnostics = e.Encode(t.Context(), "metrics_table", value)
	assert.True(t, diagnostics.HasError(), "Table grouping configuration accepted more than one property")
	tooMany["chart"].(map[string]any)["groupBy"] = "host"
	_, _, diagnostics = e.Decode(t.Context(), "metrics_table", tooMany, types.ObjectNull(nil))
	assert.True(t, diagnostics.HasError(), "Table grouping accepted the obsolete scalar form")
}

func TestDecodeNumbersAndElementArgs(t *testing.T) {
	e := testEngine(t, map[string]property{"count": testProperty("number", "chart", "count"), "fraction": testProperty("float", "chart", "fraction")})
	for _, number := range []struct {
		integer      json.Number
		wantInteger  int64
		fraction     json.Number
		wantFraction float64
	}{{"9007199254740993", 9007199254740993, "0.125", 0.125}, {"-0", 0, "0.1000000000000000001", 0.1}} {
		spec := map[string]any{"<o11y:Test>": []any{}, "chart": map[string]any{"count": number.integer, "fraction": number.fraction}}
		value, leftovers, diagnostics := e.Decode(t.Context(), "test", spec, types.ObjectNull(nil))
		require.False(t, diagnostics.HasError(), diagnostics)
		assert.Empty(t, leftovers)
		assert.Equal(t, number.wantInteger, value.Attributes()["count"].(types.Int64).ValueInt64())
		assert.Equal(t, number.wantFraction, value.Attributes()["fraction"].(types.Float64).ValueFloat64())
	}
	for _, fieldValue := range []struct {
		field string
		value any
	}{{"count", json.Number("1.1")}, {"count", json.Number("1.0")}, {"count", json.Number("1e3")}, {"count", json.Number("9223372036854775808")}, {"fraction", json.Number("1e1000")}} {
		spec := map[string]any{"<o11y:Test>": []any{}, "chart": map[string]any{fieldValue.field: fieldValue.value}, "widget": map[string]any{}, "datasource": map[string]any{}}
		_, _, d := e.Decode(context.Background(), "test", spec, types.ObjectNull(nil))
		assert.True(t, d.HasError(), "invalid numeric value accepted: %v", fieldValue.value)
	}
	spec := map[string]any{"<o11y:Test>": []any{"argument"}, "chart": map[string]any{}, "widget": map[string]any{}, "datasource": map[string]any{}}
	_, _, d := e.Decode(context.Background(), "test", spec, types.ObjectNull(nil))
	assert.True(t, d.HasError(), "element arguments accepted")
}

func TestDefinitionRejectsUnenforceableAndAmbiguousFields(t *testing.T) {
	for _, name := range []string{"id", "title"} {
		_, err := newEngine([]definition{{Type: name, Version: 1, Element: "o11y:Reserved", Properties: map[string]property{"program": testProperty("string", "datasource", "program")}}})
		assert.Error(t, err, "chart %s can overwrite a resource attribute", name)
	}
	minimum := 1
	for name, properties := range map[string]map[string]property{
		"overlap":        {"a": testProperty("string", "chart", "a"), "b": testProperty("string", "chart", "a", "b")},
		"ignored bounds": {"a": func() property { p := testProperty("bool", "chart", "a"); p.Min = &minimum; return p }()},
		"ignored values": {"a": func() property { p := testProperty("string", "chart", "a"); p.Values = []string{"allowed"}; return p }()},
		"empty values on string": {"a": func() property {
			p := testProperty("string", "chart", "a")
			p.Values = []string{}
			return p
		}()},
		"string map": {"a": func() property { p := testProperty("map", "chart", "a"); p.Item = &item{Type: "string"}; return p }()},
		"unknown condition": {"a": func() property {
			p := testProperty("string", "chart", "a")
			p.RequiredWhen = &condition{Field: "missing", Value: "x"}
			return p
		}()},
		"missing presence": {"a": {Type: "string", Description: "Test.", Mapping: mapping{Path: []string{"chart", "a"}}}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := newEngine([]definition{{Type: "test", Version: 1, Element: "T", Properties: properties}})
			assert.Error(t, err, "invalid definition accepted")
		})
	}
}

func TestOneOfRejectsUnsupportedWireAlternatives(t *testing.T) {
	for name, variants := range map[string]map[string]property{
		"bool":                 {"flag": {Type: "bool", Description: "Flag."}},
		"ambiguous strings":    {"a": {Type: "string", Description: "A."}, "b": {Type: "string", Description: "B."}},
		"required block child": {"unit": {Type: "enum", Description: "Unit.", Values: []string{"Byte"}}, "custom": {Type: "block", Properties: map[string]property{"prefix": {Type: "string", Required: true, Description: "Prefix.", Mapping: mapping{Path: []string{"prefix"}}}}}},
	} {
		t.Run(name, func(t *testing.T) {
			choice := testProperty("oneof", "chart", "choice")
			choice.Variants = variants
			_, err := newEngine([]definition{{Type: "test", Version: 1, Element: "T", Properties: map[string]property{"choice": choice}}})
			assert.Error(t, err, "unsupported oneof accepted")
		})
	}
}

func TestOneOfRejectsIgnoredBlockVariantMetadata(t *testing.T) {
	bound := 0
	for name, change := range map[string]func(*property){
		"required":     func(p *property) { p.Required = true },
		"minimum":      func(p *property) { p.Min = &bound },
		"mapping path": func(p *property) { p.Mapping.Path = []string{"custom"} },
	} {
		t.Run(name, func(t *testing.T) {
			variant := property{Type: "block", Properties: map[string]property{"prefix": testProperty("string", "prefix")}}
			change(&variant)
			choice := testProperty("oneof", "chart", "choice")
			choice.Variants = map[string]property{"custom": variant}
			_, err := newEngine([]definition{{Type: "test", Version: 1, Element: "o11y:Test", Properties: map[string]property{"choice": choice}}})
			assert.Error(t, err, "ignored block-variant metadata accepted")
		})
	}
}

func TestOneOfDecodeSelectsWireShapeAndPreservesUnmodeledFields(t *testing.T) {
	choice := testProperty("oneof", "chart", "choice")
	choice.AtLeastOneOf = []string{"unit", "prefix", "suffix"}
	choice.Variants = map[string]property{
		"unit": {Type: "enum", Description: "Named unit.", Values: []string{"Byte"}},
		"custom": {Type: "block", Properties: map[string]property{
			"prefix": testProperty("string", "prefix"),
			"suffix": testProperty("string", "suffix"),
		}},
	}
	e := testEngine(t, map[string]property{"choice": choice})
	attributes, err := e.Attributes("test")
	require.NoError(t, err)
	objectType := attributes["choice"].GetType().(types.ObjectType)
	// A prior scalar variant must not leak into a subsequent custom object.
	prior := testValue(t, e, map[string]attr.Value{"choice": types.ObjectValueMust(objectType.AttrTypes, map[string]attr.Value{"unit": types.StringValue("Byte"), "prefix": types.StringNull(), "suffix": types.StringNull()})})
	for name, test := range map[string]struct {
		wire, rebuilt any
		leftovers     []string
		unit, prefix  types.String
		invalid       bool
		warning       bool
	}{
		"scalar":                            {wire: "Byte", rebuilt: "Byte", unit: types.StringValue("Byte"), prefix: types.StringNull()},
		"object with unmodeled scalar name": {wire: map[string]any{"prefix": "$", "unit": "Byte"}, rebuilt: map[string]any{"prefix": "$"}, leftovers: []string{"chart.choice.unit"}, unit: types.StringNull(), prefix: types.StringValue("$")},
		"unmodeled scalar name alone":       {wire: map[string]any{"unit": "Byte"}, leftovers: []string{"chart.choice.unit"}, unit: types.StringNull(), prefix: types.StringNull(), warning: true},
		"invalid shape":                     {wire: true, invalid: true},
	} {
		t.Run(name, func(t *testing.T) {
			spec := map[string]any{"<o11y:Test>": []any{}, "chart": map[string]any{"choice": test.wire}}
			value, leftovers, diagnostics := e.Decode(t.Context(), "test", spec, prior)
			if test.invalid {
				assert.True(t, diagnostics.HasError(), "invalid union shape accepted")
				return
			}
			require.False(t, diagnostics.HasError(), diagnostics)
			assert.Equal(t, test.leftovers, leftovers, "incorrect union import")
			fields := value.Attributes()["choice"].(types.Object).Attributes()
			assert.True(t, fields["unit"].Equal(test.unit), "incorrect unit: expected %s, got %s", test.unit, fields["unit"])
			assert.True(t, fields["prefix"].Equal(test.prefix), "incorrect prefix: expected %s, got %s", test.prefix, fields["prefix"])
			if test.warning {
				assert.Empty(t, diagnostics, "stored union should decode cleanly")
				validation := e.Validate("test", value)
				assert.True(t, validation.HasError(), "stored union configuration should stay invalid")
				return
			}
			rebuilt, diagnostics := e.Encode(t.Context(), "test", value)
			require.False(t, diagnostics.HasError(), diagnostics)
			assert.Equal(t, test.rebuilt, rebuilt["chart"].(map[string]any)["choice"], "incorrect union reconstruction")
		})
	}
}

func TestEncodeReportsUnsupportedValues(t *testing.T) {
	t.Parallel()
	unsupported := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("A")})
	object := types.ObjectValueMust(
		map[string]attr.Type{"label": unsupported.Type(t.Context())},
		map[string]attr.Value{"label": unsupported},
	)
	children := map[string]property{"label": testProperty("string", "label")}
	block := testProperty("block", "chart", "value")
	block.Properties = children
	list := testProperty("list", "chart", "value")
	list.Item = &item{Properties: children}
	items := testProperty("map", "chart", "value")
	items.Item = list.Item
	scalarChoice := testProperty("oneof", "chart", "value")
	scalarChoice.Variants = map[string]property{"label": {Type: "string", Description: "Label."}}
	blockChoice := testProperty("oneof", "chart", "value")
	blockChoice.Variants = map[string]property{"custom": {Type: "block", Properties: children}}

	for _, tc := range []struct {
		name     string
		property property
		value    attr.Value
		location string
	}{
		{"scalar", testProperty("string", "chart", "value"), unsupported, ""},
		{"object", block, object, "label: "},
		{"list", list, types.ListValueMust(object.Type(t.Context()), []attr.Value{object}), "element 0: label: "},
		{"map", items, types.MapValueMust(object.Type(t.Context()), map[string]attr.Value{"source": object}), `key "source": label: `},
		{"scalar variant", scalarChoice, object, ""},
		{"block variant", blockChoice, object, "label: "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := testEngine(t, map[string]property{"value": tc.property})
			// Deliberately diverge from the schema to exercise the encode boundary.
			value := types.ObjectValueMust(
				map[string]attr.Type{"value": tc.value.Type(t.Context())},
				map[string]attr.Value{"value": tc.value},
			)
			spec, diagnostics := engine.Encode(t.Context(), "test", value)
			assert.Nil(t, spec, "encoding errors must not return a partial document")
			require.Len(t, diagnostics.Errors(), 1)
			assert.Equal(t, "Invalid chart value", diagnostics.Errors()[0].Summary())
			assert.Contains(t, diagnostics.Errors()[0].Detail(), "test: value: "+tc.location+"unsupported chart value")
		})
	}
}
