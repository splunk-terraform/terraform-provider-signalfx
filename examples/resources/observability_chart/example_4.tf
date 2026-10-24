resource "signalfx_observability_chart" "request_table" {
  title = "Request table"

  metrics_table = {
    program  = "A = data('requests.count').sum(by=['host']).publish('A')"
    group_by = ["host"]
    columns = [{
      field        = "A"
      header_name  = "Requests"
      display_unit = { suffix = " requests" }
    }]
  }
}
