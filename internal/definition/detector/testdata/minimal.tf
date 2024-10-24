provider "signalfx" {}

resource "signalfx_detector" "minimal" {
  name = "my minimal detector"

  program_text = <<-EOF
  detect(when(const(1) > 1)).publish('HCF')
  EOF

  rule {
    description   = "example detector"
    severity      = "Warning"
    detect_label  = "HCF"
    notifications = ["Email,test@example.com"]
  }
}
