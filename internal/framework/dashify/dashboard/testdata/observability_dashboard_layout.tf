resource "signalfx_observability_template" "dashboard_layout_chart" {
  title        = "Request rate"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = [{
      "<o11y:SingleValue>" = []
      chart                = {}
      datasource = {
        program = "A = data('requests.count').sum().publish('A')"
      }
      widget = {
        title = "Request rate"
      }
    }]
  })
}

resource "signalfx_observability_dashboard" "dashboard_layout" {
  title = "Complete layout"

  control_bar = {
    time_range = {
      label                  = "Time Range"
      description            = "Dashboard time window"
      hidden                 = false
      default_variable_value = "-PT15M"
    }

    density = {
      label                  = "Density"
      description            = "Chart resolution"
      hidden                 = false
      default_variable_value = 60
    }

    pinned_filter = [
      {
        variable_name                 = "service"
        label                         = "Service"
        description                   = "Service to inspect"
        hidden                        = false
        key                           = "service.name"
        default_variable_value        = ["checkout"]
        suggested_values              = ["checkout", "payments"]
        only_suggest_preferred_values = true
        match_missing                 = false
        required                      = true
        application_mode              = "add"
      },

      {
        variable_name          = "environment"
        key                    = "deployment.environment"
        default_variable_value = []
        application_mode       = "ignore"
      },
    ]

    filter_set = {
      label       = "Filters"
      description = "Initial ad-hoc filters"
      hidden      = false

      filter = [
        {
          key      = "deployment.environment"
          values   = ["prod"]
          negated  = false
          disabled = false
        },

        {
          key    = "region"
          values = []
        },
      ]
    }
  }

  container = [{
    template_id = signalfx_observability_template.dashboard_layout_chart.id
    layout = {
      width      = "1/2"
      height     = "20"
      min_width  = "4"
      min_height = "2"
    }
  }]
}
