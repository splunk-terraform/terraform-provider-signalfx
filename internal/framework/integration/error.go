// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"fmt"
	"net/http"

	"github.com/signalfx/signalfx-go"
)

const adminTokenHelp = "Please verify you are using an admin token when working with integrations" //nolint:gosec // User-facing guidance, not a credential.

// withAdminTokenHelp preserves the API error while adding the guidance shared by
// integrations whose write operations require an administrator session token.
func withAdminTokenHelp(err error) error {
	if err == nil {
		return nil
	}

	responseError, ok := signalfx.AsResponseError(err)
	if !ok || responseError.Code() < http.StatusBadRequest || responseError.Code() >= http.StatusInternalServerError {
		return err
	}

	return fmt.Errorf("%w\n%s", err, adminTokenHelp)
}
