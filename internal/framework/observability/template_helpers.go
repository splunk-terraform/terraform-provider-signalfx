// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/template"
)

func observabilityTemplateFromResult(result *template.Result) (*template.Template, error) {
	if result == nil || result.Data == nil {
		return nil, errors.New("template API returned no template record")
	}
	return result.Data, nil
}

func observabilityTemplateWrite(ctx context.Context, model observabilityTemplateModel) (*template.Write, diag.Diagnostics) {
	var diags diag.Diagnostics
	if model.RootElement.IsNull() || model.RootElement.IsUnknown() || model.RootElement.ValueString() == "" {
		diags.AddError("Missing template root element", "root_element must be a non-empty value.")
		return nil, diags
	}
	if err := validateObservabilityTemplateSpec(model.Spec.ValueString()); err != nil {
		diags.AddError("Invalid template specification", err.Error())
		return nil, diags
	}

	var imports []string
	var datasource *template.Datasource
	if model.Metadata != nil {
		if !model.Metadata.Imports.IsNull() && !model.Metadata.Imports.IsUnknown() {
			diags.Append(model.Metadata.Imports.ElementsAs(ctx, &imports, false)...)
			if diags.HasError() {
				return nil, diags
			}
		}

		if model.Metadata.Datasource != nil {
			datasource = &template.Datasource{
				Type:        template.DatasourceType(model.Metadata.Datasource.Type.ValueString()),
				ProgramText: model.Metadata.Datasource.ProgramText.ValueString(),
				SLOID:       model.Metadata.Datasource.SLOID.ValueString(),
			}
		}
	}

	rootElement := template.RootElement(model.RootElement.ValueString())
	return &template.Write{
		Type:  template.RecordType,
		Title: model.Title.ValueString(),
		Spec:  json.RawMessage(model.Spec.ValueString()),
		Metadata: template.WriteMetadata{
			RootElement: &rootElement,
			Imports:     imports,
			Datasource:  datasource,
		},
	}, diags
}

func observabilityTemplateModelFromRecord(prior observabilityTemplateModel, record *template.Template) (observabilityTemplateModel, error) {
	if record == nil {
		return observabilityTemplateModel{}, errors.New("template API returned no template record")
	}
	if record.ID == "" {
		return observabilityTemplateModel{}, errors.New("template API returned a template record without an ID")
	}
	if record.Metadata == nil || record.Metadata.RootElement == nil {
		return observabilityTemplateModel{}, errors.New("template API returned a template record without root element metadata")
	}
	if err := validateObservabilityTemplateSpec(string(record.Spec)); err != nil {
		return observabilityTemplateModel{}, fmt.Errorf("template API returned an invalid specification: %w", err)
	}

	var metadata *observabilityTemplateMetadataModel
	if prior.Metadata != nil {
		metadata = &observabilityTemplateMetadataModel{
			Imports:    prior.Metadata.Imports,
			Datasource: prior.Metadata.Datasource,
		}
	}

	return observabilityTemplateModel{
		ID:          types.StringValue(record.ID),
		Title:       types.StringValue(record.Title),
		RootElement: types.StringValue(string(*record.Metadata.RootElement)),
		Spec:        types.StringValue(string(record.Spec)),
		Metadata:    metadata,
	}, nil
}

func validateObservabilityTemplateSpec(raw string) error {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fmt.Errorf("spec must be valid JSON: %w", err)
	}
	if _, ok := value.(map[string]any); !ok {
		return errors.New("spec must be a JSON object")
	}
	return nil
}

func observabilityJSONEqual(a, b string) bool {
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return false
	}

	ac, err := json.Marshal(av)
	if err != nil {
		return false
	}
	bc, err := json.Marshal(bv)
	if err != nil {
		return false
	}
	return bytes.Equal(ac, bc)
}
