package tftest

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// CleanEnvVars removes any of the globally set env vars
// that are used by the library and restores them once the tests is done.
// Note this has be called before `t.Setenv` to ensure the original values are restored.
func CleanEnvVars(t *testing.T) {
	orig := os.Environ()
	for _, k := range []string{
		"SFX_AUTH_TOKEN",
		"SFX_API_URL",
	} {
		assert.NoError(t, os.Unsetenv(k), "Unable to unset environment var %q", k)
	}

	t.Cleanup(func() {
		for _, kv := range orig {
			values := strings.SplitN(kv, "=", 2)
			assert.NoError(t, os.Setenv(values[0], values[1]), "Unable to restore environment var %q", kv)
		}
	})
}
