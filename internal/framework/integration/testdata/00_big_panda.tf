resource "signalfx_big_panda_integration" "test" {
  name    = "BigPanda - My Team"
  enabled = true

  app_key = "my-app-key"
  token   = "my-token"
}
