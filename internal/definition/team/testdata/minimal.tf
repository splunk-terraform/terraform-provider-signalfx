provider "signalfx" {
  // No values are provided within tests, 
  // but refer to docs for required values to be set
}

resource "signalfx_team" "my_team" {
  provider = "signalfx" // typically not required but used within testing 

  name        = "test"
  description = "My awesome team that includes all my friends"
}