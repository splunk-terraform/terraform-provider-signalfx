// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"go.uber.org/multierr"
)

func DataSourceSchemaValidate(dataSource datasource.DataSource, model any) (errs error) {
	var resp datasource.SchemaResponse
	dataSource.Schema(context.TODO(), datasource.SchemaRequest{}, &resp)

	if resp.Schema.Description == "" && resp.Schema.MarkdownDescription == "" {
		errs = multierr.Append(errs, errors.New("missing schema description"))
	}
	if len(resp.Schema.Attributes) == 0 {
		return multierr.Combine(errs, errors.New("missing schema attribute definitions"))
	}

	expected := make(map[string]attr.Value)
	for walked := range WalkStruct(model) {
		if walked.Err() != nil {
			return walked.Err()
		}
		expected[walked.Path().String()] = walked.Attr()
	}

	actual := make(map[string]schema.Attribute)
	for path, attribute := range WalkDataSourceSchema(resp.Schema.Attributes) {
		if attribute.GetDescription() == "" && attribute.GetMarkdownDescription() == "" {
			errs = multierr.Append(errs, fmt.Errorf("field %q has no description", path.String()))
		}
		if _, ok := actual[path.String()]; ok {
			return fmt.Errorf("duplicate field in schema: %q", path.String())
		}
		actual[path.String()] = attribute
	}

	for _, field := range slices.Sorted(maps.Keys(actual)) {
		attribute := actual[field]
		value, ok := expected[field]
		if !ok {
			errs = multierr.Append(errs, fmt.Errorf("expected field not found in model: %q, check struct tags", field))
			continue
		}
		if value != nil && attribute.GetType() != value.Type(context.TODO()) {
			errs = multierr.Append(errs, fmt.Errorf("field %q has type %q, expected %q", field, attribute.GetType(), value.Type(context.TODO())))
		}
		delete(expected, field)
	}

	for field := range expected {
		errs = multierr.Append(errs, fmt.Errorf("additional field defined in model but not defined: %q", field))
	}

	return errs
}
