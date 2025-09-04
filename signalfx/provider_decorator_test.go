// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go"
	"github.com/stretchr/testify/assert"
)

func HelperValidateMethodCalled[Func schema.CreateFunc | schema.ReadFunc | schema.UpdateFunc | schema.DeleteFunc](tb testing.TB) Func {
	var called bool
	return func(data *schema.ResourceData, meta any) error {
		tb.Helper()

		called = true

		tb.Cleanup(func() {
			assert.True(tb, called, "Expected method to be called")
		})

		return nil
	}
}

func TestDeprecatedMethodDecorator(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		createSet bool
		readSet   bool
		updateSet bool
		deleteSet bool
	}{
		{
			name:      "Create method set",
			createSet: true,
		},
		{
			name:    "Read method set",
			readSet: true,
		},
		{
			name:      "Update method set",
			updateSet: true,
		},
		{
			name:      "Delete method set",
			deleteSet: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := &schema.Resource{}
			if tc.createSet {
				res.Create = HelperValidateMethodCalled[schema.CreateFunc](t)

			}
			if tc.readSet {
				res.Read = HelperValidateMethodCalled[schema.ReadFunc](t)
			}
			if tc.updateSet {
				res.Update = HelperValidateMethodCalled[schema.UpdateFunc](t)
			}
			if tc.deleteSet {
				res.Delete = HelperValidateMethodCalled[schema.DeleteFunc](t)
			}

			res = deprecatedMethodDecorator(res)

			if tc.createSet {
				assert.NoError(t, res.Create(&schema.ResourceData{}, nil))
			}

			if tc.readSet {
				assert.NoError(t, res.Read(&schema.ResourceData{}, nil))
			}
			if tc.updateSet {
				assert.NoError(t, res.Update(&schema.ResourceData{}, nil))
			}
			if tc.deleteSet {
				assert.NoError(t, res.Delete(&schema.ResourceData{}, nil))
			}
		})
	}
}

func TestWrapDeprecatedMethod(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		method schema.CreateFunc
		errVal string
	}{
		{
			name:   "No error",
			method: HelperValidateMethodCalled[schema.CreateFunc](t),
			errVal: "",
		},
		{
			name: "With error",
			method: func(data *schema.ResourceData, meta any) error {
				return &signalfx.ResponseError{}
			},
			errVal: "route \"\" had issues with status code 0\nAPI response: ",
		},
		{
			name: "With non-ResponseError",
			method: func(data *schema.ResourceData, meta any) error {
				return fmt.Errorf("some error")
			},
			errVal: "some error",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := wrapDeprecatedMethod(tc.method)(&schema.ResourceData{}, nil)
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must return the expected error")
			} else {
				assert.NoError(t, err, "Must not return an error")
			}

		})
	}
}
