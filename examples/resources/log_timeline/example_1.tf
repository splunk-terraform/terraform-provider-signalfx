resource "signalfx_log_timeline" "my_log_timeline" {
  name        = "Sample Log Timeline"
  description = "Lorem ipsum dolor sit amet, laudem tibique iracundia at mea. Nam posse dolores ex, nec cu adhuc putent honestatis"

  program_text = <<-EOF
  logs(filter=field('message') == 'Transaction processed' and field('service.name') == 'paymentservice').publish()
  EOF

  time_range = 900

}
