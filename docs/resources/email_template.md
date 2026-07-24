---
page_title: "Observability Cloud: signalfx_email_template"
description: |-
  Allows Terraform to create and manage email templates for Splunk Observability Cloud detector alerts
---
# Resource: signalfx_email_template

Email templates are reusable detector alert notification templates.

## Example

```terraform
resource "signalfx_email_template" "detector_alerts" {
  name = "Detector Alert Email"

  trigger_subject  = "Triggered: {{{detectorName}}}"
  trigger_body     = "Alert {{{messageTitle}}} triggered for {{{detectorName}}}."
  resolved_subject = "Resolved: {{{detectorName}}}"
  resolved_body    = "Alert {{{messageTitle}}} resolved for {{{detectorName}}}."

  to = ["primary@example.com"]
  cc = ["team@example.com"]

  custom_headers = {
    X-SFX-Template = "detector"
  }
}
```

## Arguments

* `name` - (Required) Name of the email template.
* `trigger_subject` - (Required) Subject used when a detector alert triggers.
* `trigger_body` - (Required) Body used when a detector alert triggers.
* `resolved_subject` - (Required) Subject used when a detector alert resolves.
* `resolved_body` - (Required) Body used when a detector alert resolves.
* `to` - (Optional) Email addresses to include as template recipients.
* `cc` - (Optional) Email addresses to include as carbon copy recipients.
* `bcc` - (Optional) Email addresses to include as blind carbon copy recipients.
* `custom_headers` - (Optional) Custom email headers to include when notifications use this template.

## Attributes

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the email template.
* `created_on_ms` - Timestamp in milliseconds when the email template was created.
* `created_by` - User that created the email template.
* `updated_on_ms` - Timestamp in milliseconds when the email template was last updated.
* `updated_by` - User that last updated the email template.
