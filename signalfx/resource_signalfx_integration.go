package signalfx

import (
	"fmt"
)

func validatePollRate(value interface{}, key string) (warns []string, errors []error) {
	v := value.(int)
	if v != 60000 && v != 300000 {
		errors = append(errors, fmt.Errorf("%q must be either 60000 or 300000, got: %d", key, v))
	}
	return
}
