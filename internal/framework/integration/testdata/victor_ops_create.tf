resource "signalfx_victor_ops_integration" "test" {
  name     = "Primary VictorOps"
  enabled  = true
  post_url = "https://alert.victorops.test/primary"
}
