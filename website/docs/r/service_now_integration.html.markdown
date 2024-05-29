---
layout: "signalfx"
page_title: "Observability Cloud: signalfx_service_now_integration"
sidebar_current: "docs-signalfx-resource-service-now-integration"
description: |-
Allows Terraform to create and manage ServiceNow Integrations for Splunk Observability Cloud
---

# Resource: signalfx_service_now_integration

ServiceNow integrations. For help with this integration see [Integration with ServiceNow](https://docs.splunk.com/observability/en/admin/notif-services/servicenow.html).

~> **NOTE** When managing integrations, use a session token of an administrator to authenticate the Observability Cloud provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example

```tf
resource "signalfx_service_now_integration" "service_now_myteam" {
  name    = "ServiceNow - My Team"
  enabled = false

  username = "thisis_me"
  password = "youd0ntsee1t"
  
  instance_name = "myinst.service-now.com"
  issue_type    = "Incident"

  alert_triggered_payload_template = "{\"short_description\": \"{{{messageTitle}}} (customized)\"}"
  alert_resolved_payload_template  = "{\"close_notes\": \"{{{messageTitle}}} (customized close msg)\"}"
}
```


## Arguments

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `username` - (Required) User name used to authenticate the ServiceNow integration.
* `password` - (Required) Password used to authenticate the ServiceNow integration.
* `instance_name` - (Required) Name of the ServiceNow instance, for example `myinst.service-now.com`.
* `issue_type` - (Required) The type of issue in standard ITIL terminology. The allowed values are `Incident` and `Problem`.
* `alert_triggered_payload_template` - (Optional) A template that Observability Cloud uses to create the ServiceNow POST JSON payloads when an alert sends a notification to ServiceNow. Use this optional field to send the values of Observability Cloud alert properties to specific fields in ServiceNow. See [API reference](https://dev.splunk.com/observability/reference/api/integrations/latest) for details.
* `alert_resolved_payload_template` - (Optional) A template that Observability Cloud uses to create the ServiceNow PUT JSON payloads when an alert is cleared in ServiceNow. Use this optional field to send the values of Observability Cloud alert properties to specific fields in ServiceNow. See [API reference](https://dev.splunk.com/observability/reference/api/integrations/latest) for details.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
