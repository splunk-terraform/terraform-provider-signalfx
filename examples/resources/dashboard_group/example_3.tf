resource "signalfx_dashboard_group" "mydashboardgroup_withmirrors" {
  name        = "My team dashboard group"
  description = "Cool dashboard group"

  // You can add as many of these as you like. Make sure your account
  // supports this feature!
  dashboard {
    dashboard_id         = signalfx_dashboard.gc_dashboard.id
    name_override        = "GC For My Service"
    description_override = "Garbage Collection dashboard maintained by JVM team"

    filter_override {
      property = "service"
      values   = ["myservice"]
      negated  = false
    }

    variable_override {
      property         = "region"
      values           = ["us-west1"]
      values_suggested = ["us-west-1", "us-east-1"]
    }
  }
}
