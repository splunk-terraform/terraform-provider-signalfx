resource "signalfx_observability_template" "test" {
  title        = "Request rate (updated)"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = []
  })

  metadata {
    imports = ["/v2/template/shared"]
  }
}
