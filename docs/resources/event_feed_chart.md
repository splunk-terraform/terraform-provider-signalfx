# Event Feed

Displays a listing of events as a widget in a dashboard.

## Example Usage

```terraform
resource "signalfx_event_feed_chart" "mynote0" {
    name = "Important Dashboard Note"
    description = "Lorem ipsum dolor sit amet"
    program_text = "A = events(eventType='Fart Testing').publish(label='A')"

    viz_options {
        label = "A"
        color = "orange"
    }
}
```


## Argument Reference

The following arguments are supported in the resource block:

* `name` - (Required) Name of the text note.
* `program_text` - (Required) Signalflow program text for the chart. More info at <https://developers.signalfx.com/docs/signalflow-overview>.
* `description` - (Optional) Description of the text note.
* `viz_options` - (Optional) Plot-level customization options, associated with a publish statement.
    * `label` - (Required) Label used in the publish statement that displays the plot (event data) you want to customize.
    * `color` - (Optional) Color to use : gray, blue, azure, navy, brown, orange, yellow, iris, magenta, pink, purple, violet, lilac, emerald, green, aquamarine. ![Colors](https://github.com/signalfx/terraform-provider-signalfx/raw/master/docs/resources/colors.png)
* `synced` - (Optional) Whether the resource in the provider and SignalFx are identical or not. Used internally for syncing, you do not need to specify it. Whenever you see a change to this field in the plan, it means that your resource has been changed from the UI and Terraform is now going to re-sync it back to what is in your configuration.
