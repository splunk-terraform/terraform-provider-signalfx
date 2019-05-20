package signalfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePollRateAllowed(t *testing.T) {
	for _, value := range []int{60000, 300000} {
		_, errors := validatePollRate(value, "poll_rate")
		assert.Equal(t, 0, len(errors))
	}
}

func TestValidatePollRateNotAllowed(t *testing.T) {
	_, errors := validatePollRate(1000, "poll_rate")
	assert.Equal(t, 1, len(errors))
}

func TestExpandServicesSucess(t *testing.T) {
	values := []interface{}{"compute", "appengine"}
	expected := []string{"compute", "appengine"}
	result := expandServices(values)
	assert.ElementsMatch(t, expected, result)
}

func TestExpandProjectServiceKeysSuccess(t *testing.T) {
	values := []interface{}{
		map[string]interface{}{
			"project_id":  "test_project",
			"project_key": "{\"type\":\"service_account\", \"project_id\": \"test_project\"}",
		},
		map[string]interface{}{
			"project_id":  "test_project_2",
			"project_key": "{\"type\":\"service_account\", \"project_id\": \"test_project_2\"}",
		},
	}
	expected := []map[string]string{
		map[string]string{
			"projectId":  "test_project",
			"projectKey": "{\"type\":\"service_account\", \"project_id\": \"test_project\"}",
		},
		map[string]string{
			"projectId":  "test_project_2",
			"projectKey": "{\"type\":\"service_account\", \"project_id\": \"test_project_2\"}",
		},
	}
	result := expandProjectServiceKeys(values)
	assert.ElementsMatch(t, expected, result)
}
