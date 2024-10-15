provider "signalfx" {}

data "signalfx_dimension_values" "os_types" {
  provider = signalfx

  query    = "os.type:*"
  order_by = "-sf_timestamp"
  limit    = 1
}
