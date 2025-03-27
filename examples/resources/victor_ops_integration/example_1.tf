resource "signalfx_victor_ops_integration" "vioctor_ops_myteam" {
  name     = "Splunk On-Call - My Team"
  enabled  = true
  post_url = "https://alert.victorops.com/integrations/generic/1234/alert/$key/$routing_key"
}
