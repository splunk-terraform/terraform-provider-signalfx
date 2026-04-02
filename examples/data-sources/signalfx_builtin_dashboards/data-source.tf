# Fetches all the of the inbuilt dashboards available to the user. 
# This can be used by users to reference the inbuilt dashboards when creating their own dashboard groups.
data "signalfx_builtin_dashboards" "example" {}

# This shows all the inbuilt dashboards available to the user.
output "all-dashboards" {
  value = data.signalfx_builtin_dashboards.example
}

## A simple example making a reference to a specific dashboard group and dashboard. 

resource "signalfx_dashboard_group" "my-service-dashboard-group" {
  name = "Example Dashboard Group"

  dashboard {
    dashboard_id  = data.signalfx_builtin_dashboards.example.results.AWS_ECS.ECS_Service
    name_override = "My Awesome Service ECS Dashboard"

    # ... Other dashboard values can be added here.
  }
}
