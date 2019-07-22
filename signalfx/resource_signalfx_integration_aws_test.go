package signalfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAwsService(t *testing.T) {
	_, errors := validateAwsService("AWS/Logs", "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateAwsService("Fart", "")
	assert.Equal(t, 1, len(errors), "Errors for invalid value")
}

func TestValidateAuthMethod(t *testing.T) {
	_, errors := validateAuthMethod("ExternalId", "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateAuthMethod("SecurityToken", "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateAuthMethod("Fart", "")
	assert.Equal(t, 1, len(errors), "Errors for invalid value")
}

func TestValidateFilterAction(t *testing.T) {
	_, errors := validateFilterAction("Exclude", "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateFilterAction("Include", "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateFilterAction("Fart", "")
	assert.Equal(t, 1, len(errors), "Errors for invalid value")
}

func TestValidateAwsPollRate(t *testing.T) {
	_, errors := validateAwsPollRate(60, "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateAwsPollRate(300, "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateAwsPollRate(12, "")
	assert.Equal(t, 1, len(errors), "Errors for invalid value")
}
