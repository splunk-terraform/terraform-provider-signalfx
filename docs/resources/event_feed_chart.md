# Text Note

This special type of chart doesn’t display any metric data. Rather, it lets you place a text note on the dashboard.

![Text Note](https://github.com/signalfx/terraform-provider-signalfx/raw/master/docs/resources/text_note.png)


## Example Usage

```terraform
resource "signalfx_event_feed_chart" "mynote0" {
    name = "Important Dashboard Note"
    description = "Lorem ipsum dolor sit amet"
    program_text = "A = events(eventType='Fart Testing').publish(label='A')"
}
```


## Argument Reference

The following arguments are supported in the resource block:

* `name` - (Required) Name of the text note.
* `program_text` - (Required) Signalflow program text for the chart. More info at <https://developers.signalfx.com/docs/signalflow-overview>.
* `description` - (Optional) Description of the text note.
* `synced` - (Optional) Whether the resource in the provider and SignalFx are identical or not. Used internally for syncing, you do not need to specify it. Whenever you see a change to this field in the plan, it means that your resource has been changed from the UI and Terraform is now going to re-sync it back to what is in your configuration.
