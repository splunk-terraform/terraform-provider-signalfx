resource "signalfx_detector" "application_delay" {
  count = length(var.clusters)

  name        = " max average delay - ${var.clusters[count.index]}"
  description = "your application is slow - ${var.clusters[count.index]}"
  max_delay   = 30
  tags        = ["app-backend", "staging"]

  # Note that if you use these features, you must use a user's
  # admin key to authenticate the provider, lest Terraform not be able
  # to modify the detector in the future!
  authorized_writer_teams = [signalfx_team.mycoolteam.id]
  authorized_writer_users = ["abc123"]

  program_text = <<-EOF
        signal = data('app.delay', filter('cluster','${var.clusters[count.index]}'), extrapolation='last_value', maxExtrapolations=5).max()
        detect(when(signal > 60, '5m')).publish('Processing old messages 5m')
        detect(when(signal > 60, '30m')).publish('Processing old messages 30m')
    EOF
  rule {
    description   = "maximum > 60 for 5m"
    severity      = "Warning"
    detect_label  = "Processing old messages 5m"
    notifications = ["Email,foo-alerts@bar.com"]
  }
  rule {
    description   = "maximum > 60 for 30m"
    severity      = "Critical"
    detect_label  = "Processing old messages 30m"
    notifications = ["Email,foo-alerts@bar.com"]
  }
}

provider "signalfx" {}

variable "clusters" {
  default = ["clusterA", "clusterB"]
}
