resource "signalfx_dashboard_group" "mydashboardgroup_withpermissions" {
  name        = "My team dashboard group"
  description = "Cool dashboard group"

  // You can add up to 25 of entries for permission configurations. 
  // Make sure your account supports this feature!
  permissions {
    principal_id    = "abc123"
    principal_type  = "ORG"
    actions         = ["READ"]
  }
  permissions {
    principal_id    = "abc456"
    principal_type  = "USER"
    actions         = ["READ", "WRITE"]
  }
}
