# Integration

SignalFx supports integrations to ingest metrics from other monitoring systems, connect to Single Sign-On providers, and to report notifications for messaging and incident management. Note that your API key must have admin permissions to use the SignalFx integration API.

## Example Usage

### PagerDuty
```terraform
resource "signalfx_integration" "pagerduty_myteam" {
    name = "PD - My Team"
    enabled = true
    type = "PagerDuty"
    api_key = "1234567890"
}
```

### GCP
```terraform
resource "signalfx_integration" "gcp_myteam" {
    name = "GCP - My Team"
    enabled = true
    type = "GCP"
    poll_rate = 300000
    services = ["compute"]
    project_service_keys = [
        {
            project_id = "gcp_project_id_1"
            project_key = "${file("/path/to/gcp_credentials_1.json")}"
        },
        {
            project_id = "gcp_project_id_2"
            project_key = "${file("/path/to/gcp_credentials_2.json")}"
        }
    ]
}
```

## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `type` - (Required) Type of the integration. See [the full list here](https://developers.signalfx.com/integrations_reference.html).
* `api_key` - (Required for `PagerDuty`) PagerDuty API key.
* `webhook_url` - (Required for `Slack`) Slack incoming webhook URL.
* `poll_rate` - (Required for `GCP`) GCP integration poll rate in milliseconds. Can be set to either 60000 or 300000 (1 minute or 5 minutes).
* `services` - (Optional for `GCP`) GCP service metrics to import. Can be an empty list, or not included, to import 'All services'.
* `project_service_keys` - (Required for `GCP`) GCP projects to add.

**Notes**

This resource does not support all known types of integration. Contributions are welcome to implement more types.

## Experimental Integration Specific Resources

In a future release the current generic `signalfx_integration` will be replaced with specific resources for each integration. The first instance of this, `signalfx_pagerduty_integration` provides an example.

```terraform
resource "signalfx_integration" "signalfx_pagerduty_integration" {
    name = "PD - My Team"
    enabled = true
    api_key = "1234567890"
}
```
