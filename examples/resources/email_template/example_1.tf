resource "signalfx_email_template" "detector_alerts" {
  name = "Detector Alert Email"

  trigger_subject  = "Triggered: {{{detectorName}}}"
  trigger_body     = "Alert {{{messageTitle}}} triggered for {{{detectorName}}}."
  resolved_subject = "Resolved: {{{detectorName}}}"
  resolved_body    = "Alert {{{messageTitle}}} resolved for {{{detectorName}}}."

  to = ["primary@example.com"]
  cc = ["team@example.com"]

  custom_headers = {
    X-SFX-Template = "detector"
  }
}
