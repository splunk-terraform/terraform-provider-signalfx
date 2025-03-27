resource "signalfx_dashboard" "mydashboard_inheritingpermissions" {
  name            = "My Dashboard"
  dashboard_group = signalfx_dashboard_group.mydashboardgroup0.id
  
  // Make sure your account supports this feature!
  permissions {
    parent = signalfx_dashboard_group.mydashboardgroup0.id 
  }
  // ...
}
