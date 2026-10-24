resource "signalfx_observability_chart" "request_total" {
  title = "Request total"

  metrics_single_value = {
    program                 = "A = data('requests.count').sum().publish('A')"
    time_range              = "15m"
    maximum_fraction_digits = 0
    display_unit            = { suffix = " requests" }
  }
}
