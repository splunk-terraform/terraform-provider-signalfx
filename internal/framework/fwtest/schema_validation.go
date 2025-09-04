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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"go.uber.org/multierr"
)

func ResourceSchemaValidate(res resource.Resource, model any) (errs error) {
	var resp resource.SchemaResponse
	res.Schema(context.TODO(), resource.SchemaRequest{}, &resp)

	if resp.Schema.Description == "" {
		errs = multierr.Append(errs, errors.New("missing schema description"))
	}

	if len(resp.Schema.Attributes) == 0 {
		return multierr.Combine(errs, errors.New("missing schema attribute definitions"))
	}

	expected := make(map[string]attr.Value)
	for wv := range WalkStruct(model) {
		if wv.Err() != nil {
			return wv.Err()
		}
		expected[wv.Path().String()] = wv.Attr()
	}

	actual := make(map[string]schema.Attribute)
	for p, attr := range WalkResourceSchema(resp.Schema.Attributes) {
		if attr.GetDescription() == "" && attr.GetMarkdownDescription() == "" {
			errs = multierr.Append(errs, fmt.Errorf("field %q has no description", p.String()))
		}
		if _, ok := actual[p.String()]; ok {
			return fmt.Errorf("duplicate field in schema: %q", p.String())
		}
		actual[p.String()] = attr
	}

	for _, field := range slices.Sorted(maps.Keys(actual)) {
		actual := actual[field]
		t, ok := expected[field]
		if !ok {
			errs = multierr.Append(errs, fmt.Errorf("expected field not found in model: %q, check struct tags", field))
			continue
		}
		if t != nil && actual.GetType() != t.Type(context.TODO()) {
			errs = multierr.Append(errs, fmt.Errorf("field %q has type %q, expected %q", field, actual.GetType(), t.Type(context.TODO())))
		}
		delete(expected, field)
	}

	if len(expected) > 0 {
		for field := range expected {
			errs = multierr.Append(errs, fmt.Errorf("additional field defined in model but not defined: %q", field))
		}
	}

	return errs
}
