resource "signalfx_victor_ops_integration" "test" {
  name     = "Updated VictorOps"
  enabled  = false
  post_url = "https://alert.victorops.test/updated"
}
