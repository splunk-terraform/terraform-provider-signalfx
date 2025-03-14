resource "signalfx_org_token" "minimal" {
  name        = "My Token"
  description = "This is my token"
  auth_scopes = ["API"]
}
