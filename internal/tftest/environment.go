// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// CleanEnvVars removes any of the globally set env vars
// that are used by the library and restores them once the tests is done.
// Note this has be called before `t.Setenv` to ensure the original values are restored.
func CleanEnvVars(t *testing.T) {
	orig := make(map[string]string)
	for _, k := range []string{
		"SFX_AUTH_TOKEN",
		"SFX_API_URL",
	} {
		if v, ok := os.LookupEnv(k); ok {
			orig[k] = v
			assert.NoError(t, os.Unsetenv(k), "Unable to unset environment var %q", k)
		}
	}

	t.Cleanup(func() {
		for k, v := range orig {
			assert.NoError(t, os.Setenv(k, v), "Unable to restore environment var %s=%s", k, v)
		}
	})
}
