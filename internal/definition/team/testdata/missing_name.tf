provider "signalfx" {
  // No values are provided within tests, 
  // but refer to docs for required values to be set
}

resource "signalfx_team" "my-test-team" {
  provider = "signalfx" // typically not required but used within testing 
}