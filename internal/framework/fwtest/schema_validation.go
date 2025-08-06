// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return multierr.Combine(errs, fmt.Errorf("model must be a struct, provided: %T", model))
	}

	expected := make(map[string]attr.Value)
	for i := range v.Type().NumField() {
		var (
			field = v.Type().Field(i)
			named = field.Tag.Get("tfsdk")
		)
		if named == "" {
			continue
		}
		if !field.IsExported() {
			errs = multierr.Append(errs, fmt.Errorf("framework requires field exported %q", field.Name))
			continue
		}
		// TODO(MovieStoreGuy): This is fine until we handle more complex types.
		switch f := reflect.Zero(field.Type).Interface().(type) {
		case attr.Value:
			expected[named] = f
		default:
			// Handle the case where the field is potentially nested or complex.
			panic("You have entered the twilight zone. Please check the model definition.")
		}
	}

	for field, actual := range resp.Schema.Attributes {
		t, ok := expected[field]
		if !ok {
			errs = multierr.Append(errs, fmt.Errorf("expected field not found in model: %q, check struct tags", field))
			continue
		}
		if actual.GetType() != t.Type(context.TODO()) {
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
