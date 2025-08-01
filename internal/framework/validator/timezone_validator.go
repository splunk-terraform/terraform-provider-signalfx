package fwvalidator

import (
	"context"
	"time"
	_ "time/tzdata" // Importing time zone database to ensure there is failover option

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Timezone struct{}

var (
	_ validator.String = (*Timezone)(nil)
)

func NewTimeZoneValidator() Timezone {
	return Timezone{}
}

func (Timezone) Description(_ context.Context) string {
	return "Validates that a string is a valid UTC time format."
}

func (tz Timezone) MarkdownDescription(ctx context.Context) string {
	return tz.Description(ctx)
}

func (Timezone) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	val := types.StringNull()

	if details := req.Config.GetAttribute(ctx, req.Path, &val); details.HasError() {
		resp.Diagnostics.Append(details...)
		return
	}

	if _, err := time.LoadLocation(val.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Time Zone",
			err.Error(),
		)
		return
	}
}
