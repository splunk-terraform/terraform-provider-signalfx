// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package interpreter

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func testProperty(kind string, path ...string) Property {
	return Property{Type: kind, Optional: true, Description: "Test field.", Mapping: Mapping{Path: path}}
}
func testEngine(t *testing.T, properties map[string]Property) *Engine {
	t.Helper()
	e, err := newEngine([]Definition{{Type: "test", Version: 1, Element: "o11y:Test", Properties: properties}})
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func testValue(t *testing.T, e *Engine, overrides map[string]attr.Value) types.Object {
	t.Helper()
	attributes, _ := e.Attributes("test")
	attributeTypes := AttributeTypes(attributes)
	values := map[string]attr.Value{}
	for name, typ := range attributeTypes {
		value, err := typ.ValueFromTerraform(t.Context(), tftypes.NewValue(typ.TerraformType(t.Context()), nil))
		if err != nil {
			t.Fatal(err)
		}
		values[name] = value
	}
	for name, value := range overrides {
		values[name] = value
	}
	value, d := types.ObjectValue(attributeTypes, values)
	if d.HasError() {
		t.Fatal(d)
	}
	return value
}

func TestValidationDefersUnknownAndEnforcesConditionalCollections(t *testing.T) {
	mode := testProperty("enum", "chart", "mode")
	mode.Values = []string{"Scale", "None"}
	scale := testProperty("list", "chart", "scale")
	scale.Item = &Item{Type: "string"}
	scale.RequiredWhen = &Condition{Field: "mode", Value: "Scale"}
	scale.RequiresValue = &Condition{Field: "mode", Value: "Scale"}
	e := testEngine(t, map[string]Property{"mode": mode, "scale": scale})
	unknown := testValue(t, e, map[string]attr.Value{"mode": types.StringValue("Scale"), "scale": types.ListUnknown(types.StringType)})
	if d := e.Validate(context.Background(), "test", unknown, false); d.HasError() {
		t.Fatal("unknown rejected during plan", d)
	}
	if d := e.Validate(context.Background(), "test", unknown, true); !d.HasError() {
		t.Fatal("unknown accepted during apply")
	}
	empty := testValue(t, e, map[string]attr.Value{"mode": types.StringValue("Scale"), "scale": types.ListValueMust(types.StringType, []attr.Value{})})
	if d := e.Validate(context.Background(), "test", empty, false); !d.HasError() {
		t.Fatal("conditional empty collection accepted")
	}
	wrong := testValue(t, e, map[string]attr.Value{"mode": types.StringValue("None"), "scale": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("red")})})
	if d := e.Validate(context.Background(), "test", wrong, false); !d.HasError() {
		t.Fatal("required sibling value ignored")
	}
}

func TestUnicodeLengthAndFiniteFloats(t *testing.T) {
	minimum := 2
	title := testProperty("string", "widget", "title")
	title.MinLength = &minimum
	e := testEngine(t, map[string]Property{"title": title, "limit": testProperty("float", "chart", "limit")})
	for _, limit := range []float64{math.Inf(1), math.Inf(-1)} {
		value := testValue(t, e, map[string]attr.Value{"title": types.StringValue("你好"), "limit": types.Float64Value(limit)})
		if _, d := e.Encode(context.Background(), "test", value); !d.HasError() {
			t.Fatal("nonfinite write accepted")
		}
	}
	value := testValue(t, e, map[string]attr.Value{"title": types.StringValue("你")})
	if d := e.Validate(context.Background(), "test", value, false); !d.HasError() {
		t.Fatal("UTF-8 bytes counted as characters")
	}
}

func TestNestedCollectionsRoundTrip(t *testing.T) {
	label := testProperty("string", "label")
	list := testProperty("list", "chart", "windows")
	list.Item = &Item{Properties: map[string]Property{"label": label}}
	items := testProperty("map", "chart", "namedWindows")
	items.Item = list.Item
	e := testEngine(t, map[string]Property{"windows": list, "named_windows": items})
	attributes, _ := e.Attributes("test")
	objectType := attributes["windows"].GetType().(types.ListType).ElemType.(types.ObjectType)
	window := types.ObjectValueMust(objectType.AttrTypes, map[string]attr.Value{"label": types.StringValue("sample")})
	configured := testValue(t, e, map[string]attr.Value{"windows": types.ListValueMust(objectType, []attr.Value{window}), "named_windows": types.MapValueMust(objectType, map[string]attr.Value{"a": window})})
	spec, d := e.Encode(context.Background(), "test", configured)
	if d.HasError() {
		t.Fatal(d)
	}
	decoded, leftovers, d := e.Decode(context.Background(), "test", spec, types.ObjectNull(nil))
	if d.HasError() || len(leftovers) > 0 || !decoded.Equal(configured) {
		t.Fatalf("nested collections changed: %v %v", leftovers, d)
	}
	empty := testValue(t, e, map[string]attr.Value{"windows": types.ListValueMust(objectType, []attr.Value{}), "named_windows": types.MapValueMust(objectType, map[string]attr.Value{})})
	spec, d = e.Encode(context.Background(), "test", empty)
	if d.HasError() {
		t.Fatal(d)
	}
	decoded, leftovers, d = e.Decode(context.Background(), "test", spec, empty)
	if d.HasError() || len(leftovers) > 0 || !decoded.Equal(empty) {
		t.Fatal("empty collections lost", leftovers, d)
	}
}

func TestTableGroupingDecodesCurrentArrays(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
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
			if diagnostics.HasError() || len(leftovers) != 0 {
				t.Fatalf("decode: leftovers %v, diagnostics %v", leftovers, diagnostics)
			}
			grouping := value.Attributes()["group_by"].(types.List)
			if grouping.IsNull() || !grouping.Equal(types.ListValueMust(types.StringType, test.want)) {
				t.Fatalf("unexpected grouping: %v", grouping)
			}
			encoded, diagnostics := e.Encode(t.Context(), "metrics_table", value)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if got := encoded["chart"].(map[string]any)["groupBy"].([]any); !reflect.DeepEqual(got, test.wantWire) {
				t.Fatalf("encoded grouping %v, want %v", got, test.wantWire)
			}
		})
	}
	tooMany := map[string]any{
		"<o11y:TableChart>": []any{},
		"widget":            map[string]any{},
		"datasource":        map[string]any{"program": "A"},
		"chart":             map[string]any{"groupBy": []any{"host", "region"}},
	}
	value, _, diagnostics := e.Decode(t.Context(), "metrics_table", tooMany, types.ObjectNull(nil))
	if diagnostics.HasError() || len(diagnostics) != 0 {
		t.Fatal("stored grouping outside the configured limit should decode without diagnostics", diagnostics)
	}
	if _, diagnostics := e.Encode(t.Context(), "metrics_table", value); !diagnostics.HasError() {
		t.Fatal("Table grouping configuration accepted more than one property")
	}
	tooMany["chart"].(map[string]any)["groupBy"] = "host"
	if _, _, diagnostics := e.Decode(t.Context(), "metrics_table", tooMany, types.ObjectNull(nil)); !diagnostics.HasError() {
		t.Fatal("Table grouping accepted the obsolete scalar form")
	}
}

func TestDecodeRejectsUnsafeNumericsAndElementArgs(t *testing.T) {
	e := testEngine(t, map[string]Property{"count": testProperty("number", "chart", "count"), "fraction": testProperty("float", "chart", "fraction")})
	for _, number := range []struct {
		wire json.Number
		want int64
	}{{"9007199254740993", 9007199254740993}, {"1.0", 1}, {"1e0", 1}, {"-0", 0}} {
		spec := map[string]any{"<o11y:Test>": []any{}, "chart": map[string]any{"count": number.wire, "fraction": json.Number("0.125")}}
		value, leftovers, diagnostics := e.Decode(t.Context(), "test", spec, types.ObjectNull(nil))
		if diagnostics.HasError() || len(leftovers) != 0 || value.Attributes()["count"].(types.Int64).ValueInt64() != number.want || value.Attributes()["fraction"].(types.Float64).ValueFloat64() != 0.125 {
			t.Fatal("exact numeric value changed", value, leftovers, diagnostics)
		}
	}
	for _, fieldValue := range []struct {
		field string
		value any
	}{{"count", json.Number("1.1")}, {"count", json.Number("9223372036854775808")}, {"fraction", json.Number("0.1000000000000000001")}, {"fraction", math.NaN()}} {
		spec := map[string]any{"<o11y:Test>": []any{}, "chart": map[string]any{fieldValue.field: fieldValue.value}, "widget": map[string]any{}, "datasource": map[string]any{}}
		if _, _, d := e.Decode(context.Background(), "test", spec, types.ObjectNull(nil)); !d.HasError() {
			t.Fatalf("unsafe number accepted: %v", fieldValue.value)
		}
	}
	spec := map[string]any{"<o11y:Test>": []any{"argument"}, "chart": map[string]any{}, "widget": map[string]any{}, "datasource": map[string]any{}}
	if _, _, d := e.Decode(context.Background(), "test", spec, types.ObjectNull(nil)); !d.HasError() {
		t.Fatal("element arguments accepted")
	}
}

func TestDefinitionRejectsUnenforceableAndAmbiguousFields(t *testing.T) {
	for _, name := range []string{"id", "title"} {
		_, err := newEngine([]Definition{{Type: name, Version: 1, Element: "o11y:Reserved", Properties: map[string]Property{"program": testProperty("string", "datasource", "program")}}})
		if err == nil {
			t.Fatalf("chart %s can overwrite a resource attribute", name)
		}
	}
	minimum := 1
	for name, properties := range map[string]map[string]Property{
		"overlap":        {"a": testProperty("string", "chart", "a"), "b": testProperty("string", "chart", "a", "b")},
		"ignored bounds": {"a": func() Property { p := testProperty("bool", "chart", "a"); p.Min = &minimum; return p }()},
		"ignored values": {"a": func() Property { p := testProperty("string", "chart", "a"); p.Values = []string{"allowed"}; return p }()},
		"empty values on string": {"a": func() Property {
			p := testProperty("string", "chart", "a")
			p.Values = []string{}
			return p
		}()},
		"string map": {"a": func() Property { p := testProperty("map", "chart", "a"); p.Item = &Item{Type: "string"}; return p }()},
		"unknown condition": {"a": func() Property {
			p := testProperty("string", "chart", "a")
			p.RequiredWhen = &Condition{Field: "missing", Value: "x"}
			return p
		}()},
		"missing presence": {"a": {Type: "string", Description: "Test.", Mapping: Mapping{Path: []string{"chart", "a"}}}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := newEngine([]Definition{{Type: "test", Version: 1, Element: "T", Properties: properties}})
			if err == nil {
				t.Fatal("invalid definition accepted")
			}
		})
	}
}

func TestOneOfRejectsUnsupportedWireAlternatives(t *testing.T) {
	for name, variants := range map[string]map[string]Property{
		"bool":                 {"flag": {Type: "bool", Description: "Flag."}},
		"ambiguous strings":    {"a": {Type: "string", Description: "A."}, "b": {Type: "string", Description: "B."}},
		"required block child": {"unit": {Type: "enum", Description: "Unit.", Values: []string{"Byte"}}, "custom": {Type: "block", Properties: map[string]Property{"prefix": {Type: "string", Required: true, Description: "Prefix.", Mapping: Mapping{Path: []string{"prefix"}}}}}},
	} {
		t.Run(name, func(t *testing.T) {
			choice := testProperty("oneof", "chart", "choice")
			choice.Variants = variants
			_, err := newEngine([]Definition{{Type: "test", Version: 1, Element: "T", Properties: map[string]Property{"choice": choice}}})
			if err == nil {
				t.Fatal("unsupported oneof accepted")
			}
		})
	}
}

func TestOneOfRejectsIgnoredBlockVariantMetadata(t *testing.T) {
	bound := 0
	for name, change := range map[string]func(*Property){
		"reference":          func(p *Property) { p.Ref = "common/fields.yml#display_unit" },
		"optional":           func(p *Property) { p.Optional = true },
		"required":           func(p *Property) { p.Required = true },
		"values":             func(p *Property) { p.Values = []string{"Byte"} },
		"empty values":       func(p *Property) { p.Values = []string{} },
		"default":            func(p *Property) { p.Default = false },
		"minimum":            func(p *Property) { p.Min = &bound },
		"maximum":            func(p *Property) { p.Max = &bound },
		"minimum length":     func(p *Property) { p.MinLength = &bound },
		"maximum items":      func(p *Property) { p.MaxItems = &bound },
		"conflicts":          func(p *Property) { p.ConflictsWith = []string{"unit"} },
		"empty conflicts":    func(p *Property) { p.ConflictsWith = []string{} },
		"at least one":       func(p *Property) { p.AtLeastOneOf = []string{"prefix"} },
		"empty at least one": func(p *Property) { p.AtLeastOneOf = []string{} },
		"requires value":     func(p *Property) { p.RequiresValue = &Condition{Field: "unit", Value: "Byte"} },
		"required when":      func(p *Property) { p.RequiredWhen = &Condition{Field: "unit", Value: "Byte"} },
		"mapping path":       func(p *Property) { p.Mapping.Path = []string{"custom"} },
		"empty mapping path": func(p *Property) { p.Mapping.Path = []string{} },
		"mapping transform":  func(p *Property) { p.Mapping.Transform = "negate" },
		"item":               func(p *Property) { p.Item = &Item{} },
		"variants":           func(p *Property) { p.Variants = map[string]Property{"unit": testProperty("string", "unit")} },
		"empty variants":     func(p *Property) { p.Variants = map[string]Property{} },
	} {
		t.Run(name, func(t *testing.T) {
			variant := Property{Type: "block", Properties: map[string]Property{"prefix": testProperty("string", "prefix")}}
			change(&variant)
			choice := testProperty("oneof", "chart", "choice")
			choice.Variants = map[string]Property{"custom": variant}
			if _, err := newEngine([]Definition{{Type: "test", Version: 1, Element: "o11y:Test", Properties: map[string]Property{"choice": choice}}}); err == nil {
				t.Fatal("ignored block-variant metadata accepted")
			}
		})
	}
}

func TestOneOfDecodeSelectsWireShapeAndPreservesUnmodeledFields(t *testing.T) {
	choice := testProperty("oneof", "chart", "choice")
	choice.AtLeastOneOf = []string{"unit", "prefix", "suffix"}
	choice.Variants = map[string]Property{
		"unit": {Type: "enum", Description: "Named unit.", Values: []string{"Byte"}},
		"custom": {Type: "block", Properties: map[string]Property{
			"prefix": testProperty("string", "prefix"),
			"suffix": testProperty("string", "suffix"),
		}},
	}
	e := testEngine(t, map[string]Property{"choice": choice})
	attributes, _ := e.Attributes("test")
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
				if !diagnostics.HasError() {
					t.Fatal("invalid union shape accepted")
				}
				return
			}
			if diagnostics.HasError() || !reflect.DeepEqual(leftovers, test.leftovers) {
				t.Fatal("incorrect union import", leftovers, diagnostics)
			}
			fields := value.Attributes()["choice"].(types.Object).Attributes()
			if !fields["unit"].Equal(test.unit) || !fields["prefix"].Equal(test.prefix) {
				t.Fatal("inactive variant claimed fields", fields)
			}
			if test.warning {
				if len(diagnostics) != 0 || !e.Validate(t.Context(), "test", value, true).HasError() {
					t.Fatal("stored union should decode cleanly while configuration stays invalid", diagnostics)
				}
				return
			}
			rebuilt, diagnostics := e.Encode(t.Context(), "test", value)
			if diagnostics.HasError() || !reflect.DeepEqual(rebuilt["chart"].(map[string]any)["choice"], test.rebuilt) {
				t.Fatal("incorrect union reconstruction", rebuilt, diagnostics)
			}
		})
	}
}

func TestEngineReturnsIndependentSchemaViews(t *testing.T) {
	minimum := 1
	name := testProperty("string", "name")
	name.MinLength = &minimum
	style := testProperty("enum", "style")
	style.Values = []string{"line", "area"}
	options := testProperty("block", "chart", "options")
	options.Properties = map[string]Property{"name": name, "style": style}
	properties := map[string]Property{"options": options}
	e := testEngine(t, properties)
	attributes, _ := e.Attributes("test")
	objectType := attributes["options"].GetType().(types.ObjectType)
	value := testValue(t, e, map[string]attr.Value{"options": types.ObjectValueMust(objectType.AttrTypes, map[string]attr.Value{"name": types.StringValue("x"), "style": types.StringValue("line")})})

	// Changes to a returned schema must not affect the shared engine.
	view := attributes["options"].(schema.SingleNestedAttribute)
	delete(view.Attributes, "style")
	delete(attributes, "options")

	if d := e.Validate(t.Context(), "test", value, true); d.HasError() {
		t.Fatal("caller changed the engine's validation contract", d)
	}
	spec, d := e.Encode(t.Context(), "test", value)
	if d.HasError() || spec["chart"] == nil || spec["other"] != nil {
		t.Fatal("caller changed the engine's wire mappings", spec, d)
	}
	fresh, _ := e.Attributes("test")
	if len(fresh["options"].(schema.SingleNestedAttribute).Attributes) != 2 {
		t.Fatal("caller changed the engine's schema")
	}
	wrong := types.ObjectValueMust(map[string]attr.Type{"name": types.StringType}, map[string]attr.Value{"name": types.StringValue("x")})
	if d := e.Validate(t.Context(), "test", wrong, false); !d.HasError() {
		t.Fatal("mismatched Terraform object type accepted")
	}
}
