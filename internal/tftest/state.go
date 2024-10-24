package tftest

import (
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// DelayStateCheck should be used in times when the services
// need some additional processing time to ensure that all the expected
// values are populated.
func DelayStateCheck(delay time.Duration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		<-time.After(delay)
		return nil
	}
}
