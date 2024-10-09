package check

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func Notification() schema.SchemaValidateDiagFunc {
	return func(i any, p cty.Path) diag.Diagnostics {
		s, ok := i.(string)
		if !ok {
			return tfext.AsErrorDiagnostics(
				fmt.Errorf("expected %v to be of type string", i),
				p,
			)
		}
		// Using the helper library to avoid repeating code
		_, err := common.NewNotificationFromString(s)
		if err == nil {
			return nil
		}
		return tfext.AsErrorDiagnostics(err, p)
	}
}
