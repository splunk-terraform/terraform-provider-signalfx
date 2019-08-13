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
