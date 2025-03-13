package orgtoken

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewResource(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewResource(), "Must return a valid value")
}
