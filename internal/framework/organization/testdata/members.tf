data "signalfx_organization_members" "test" {
  emails = [
    "alice@example.com",
    "bob@example.com",
  ]
}
