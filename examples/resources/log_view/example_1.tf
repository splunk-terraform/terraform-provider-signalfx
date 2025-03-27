resource "signalfx_log_view" "my_log_view" {
  name        = "Sample Log View"
  description = "Lorem ipsum dolor sit amet, laudem tibique iracundia at mea. Nam posse dolores ex, nec cu adhuc putent honestatis"

  program_text = <<-EOF
  logs(filter=field('message') == 'Transaction processed' and field('service.name') == 'paymentservice').publish()
  EOF

  time_range = 900
  sort_options {
    descending= false
    field= "severity"
   }

  columns {
        name="severity"
    }
  columns {
        name="time"
    }
  columns {
        name="amount.currency_code"
    }
  columns {
        name="amount.nanos"
    }
  columns {
        name="amount.units"
    }
  columns {
        name="message"
    }

}
