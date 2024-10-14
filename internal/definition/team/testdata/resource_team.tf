provider "signalfx" {}

resource "signalfx_team" "example_test" {
  provider = signalfx

  name        = "my team"
  description = "An example of team"
}
