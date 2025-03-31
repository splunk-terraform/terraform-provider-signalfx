resource "signalfx_event_feed_chart" "mynote0" {
  name         = "Important Dashboard Note"
  description  = "Lorem ipsum dolor sit amet"
  program_text = "A = events(eventType='My Event Type').publish(label='A')"

  viz_options {
    label = "A"
    color = "orange"
  }
}
