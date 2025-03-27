resource "signalfx_dashboard" "mydashboard_custompermissions" {
  name            = "My Dashboard"
  dashboard_group = signalfx_dashboard_group.mydashboardgroup0.id

  // You can add up to 25 of entries for permission configurations.
  // Make sure your account supports this feature!
  permissions {
    acl {
      principal_id    = "abc123"
      principal_type  = "ORG"
      actions         = ["READ"]
    }
    acl {
      principal_id    = "abc456"
      principal_type  = "USER"
      actions         = ["READ", "WRITE"]
    }
  }
  // ...
}
