// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go"
)

func deprecatedMethodDecorator(res *schema.Resource) *schema.Resource {
	if res == nil {
		return nil
	}

	if res.Create != nil {
		res.Create = wrapDeprecatedMethod(res.Create)
	}

	if res.Read != nil {
		res.Read = wrapDeprecatedMethod(res.Read)
	}

	if res.Update != nil {
		res.Update = wrapDeprecatedMethod(res.Update)
	}

	if res.Delete != nil {
		res.Delete = wrapDeprecatedMethod(res.Delete)
	}

	return res
}

func wrapDeprecatedMethod[Func schema.CreateFunc | schema.UpdateFunc | schema.ReadFunc | schema.DeleteFunc](fn Func) Func {
	return func(data *schema.ResourceData, meta any) error {
		err := fn(data, meta)
		if err == nil {
			return nil
		}

		rerr, ok := signalfx.AsResponseError(err)
		if !ok {
			return err
		}
		// HACK(MovieStoreGuy):
		// By default, if a resource is externally deleted,
		// it should be removed from the state file
		// Since the deprecated methods treat a returned error as a failures,
		// the error is suppressed here and should encourage the user to try again.
		if rerr.Code() == http.StatusNotFound {
			data.SetId("")
			return nil
		}

		// Include the API response details as a part of the returned error
		// TODO: Remove this once the deprecated methods are removed
		return fmt.Errorf("%w\nAPI response: %s", err, rerr.Details())
	}
}
