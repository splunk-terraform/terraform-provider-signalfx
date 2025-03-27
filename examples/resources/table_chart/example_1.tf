# signalfx_list_chart.Logs-Exec_0:
resource "signalfx_table_chart" "table_0" {
    description             = "beep"
    disable_sampling        = false
    max_delay               = 0
    name                    = "TableChart!"
    program_text            = "A = data('cpu.usage.total').publish(label='CPU Total')"
    group_by                = ["ClusterName"]
}
