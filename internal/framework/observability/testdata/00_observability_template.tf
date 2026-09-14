resource "signalfx_observability_template" "test" {
  title        = "Request rate"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = []
  })
}
