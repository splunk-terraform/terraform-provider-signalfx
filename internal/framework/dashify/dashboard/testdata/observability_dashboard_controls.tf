resource "signalfx_observability_dashboard" "dashboard_controls" {
  title = "Service health"

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
        property                      = "service.name"
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
    template_id = "chart-template-id"
  }]
}
