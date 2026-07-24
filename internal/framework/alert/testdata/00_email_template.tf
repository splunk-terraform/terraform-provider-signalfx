resource "signalfx_email_template" "test" {
  name = "Detector Alert Email"

  trigger_subject  = "Triggered: {{{detectorName}}}"
  trigger_body     = "Alert body {{{messageTitle}}}"
  resolved_subject = "Resolved: {{{detectorName}}}"
  resolved_body    = "Resolved body {{{messageTitle}}}"

  to  = ["primary@example.com"]
  cc  = ["team@example.com"]
  bcc = ["audit@example.com"]

  custom_headers = {
    X-SFX-Template = "detector"
  }
}
