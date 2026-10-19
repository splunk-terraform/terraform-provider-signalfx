variable "filter_property" {
  type    = string
  default = "service.name"
}

resource "signalfx_observability_template" "chart" {
  title        = "Request rate"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = []
  })
}

resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"

  control_bar = {
    time_range = {
      label       = "Time Range"
      description = "Dashboard time window"
      default     = "-PT15M"
    }

    density = {
      label       = "Density"
      description = "Chart resolution"
      default     = 60
    }

    pinned_filter = [
      {
        label                         = "Service"
        description                   = "Service to inspect"
        property                      = var.filter_property
        default                       = ["checkout"]
        suggested_values              = ["checkout", "payments"]
        only_suggest_preferred_values = true
        match_missing                 = false
        required                      = true
        application_mode              = "add"
      },

      {
        variable_name    = "environment"
        property         = "deployment.environment"
        default          = []
        application_mode = "ignore"
      },
    ]

    filter_set = {
      label       = "Filters"
      description = "Initial ad-hoc filters"

      filter = [
        {
          property = "deployment.environment"
          values   = ["prod"]
          negated  = false
          disabled = false
        },

        {
          property = "region"
          values   = []
        },
      ]
    }
  }

  container = [{
    template_id = signalfx_observability_template.chart.id
    layout = {
      x          = "0"
      width      = "1/2"
      height     = "20"
      min_width  = "4"
      min_height = "2"
    }
  }]
}
