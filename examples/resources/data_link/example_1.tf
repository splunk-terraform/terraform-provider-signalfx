# A global link to Splunk Observability Cloud dashboard.
resource "signalfx_data_link" "my_data_link" {
  property_name  = "pname"
  property_value = "pvalue"

  target_signalfx_dashboard {
    is_default         = true
    name               = "sfx_dash"
    dashboard_group_id = signalfx_dashboard_group.mydashboardgroup0.id
    dashboard_id       = signalfx_dashboard.mydashboard0.id
  }
}

# A dashboard-specific link to an external URL
resource "signalfx_data_link" "my_data_link_dash" {
  context_dashboard_id = signalfx_dashboard.mydashboard0.id
  property_name        = "pname2"
  property_value       = "pvalue"

  target_external_url {
    name        = "ex_url"
    time_format = "ISO8601"
    url         = "https://www.example.com"
    property_key_mapping = {
      foo = "bar"
    }
  }
}

# A link to an AppDynamics Service
resource "signalfx_data_link" "my_data_link_appd" {
  property_name        = "pname3"
  property_value       = "pvalue"

  target_appd_url {
    name        = "appd_url"
    url         = "https://www.example.saas.appdynamics.com/#/application=1234&component=5678"
  }
}
