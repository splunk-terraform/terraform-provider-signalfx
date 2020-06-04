---
layout: "signalfx"
page_title: "SignalFx: signalfx_aws_services"
sidebar_current: "docs-signalfx-signalfx-aws-services"
description: |-
  Provides a list AWS service names.
---

# Data Source: signalfx_aws_services

Use this data source to get a list of AWS service names.

## Example Usage

```hcl
data "signalfx_aws_services" "aws_services" {
}

# Leaves out most of the integration bits, see the docs
# for signalfx_aws_integration for more
resource "signalfx_aws_integration" "aws_myteam" {
  # …

  # All supported services!
  services = data.signalfx_aws_services.aws_services.services.*.name
}
```

## Argument Reference

None

## Attributes Reference

`services` is set to all available service names supported by SignalFx
