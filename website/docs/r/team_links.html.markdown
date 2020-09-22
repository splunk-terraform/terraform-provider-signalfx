---
layout: "signalfx"
page_title: "SignalFx: signalfx_resource"
sidebar_current: "docs-signalfx-resource-team-links"
description: |-
  Allows Terraform to link dashboard groups and detectors to teams.
---

# Resource: signalfx_team_links

Handles management of linking dashboard groups and detectors to SignalFx teams. Multiple links can link to the same team but they should link to unique detectors and dashboard groups.

## Example Usage

```tf
resource "signalfx_team" "prod" {
  name        = "prod"
  description = "Production on-call"
}

resource "signalfx_dashboard_group" "infra" {
    name = "infra"
    description = "infra dashboard"
}

resource "signalfx_team_links" "link1" {
  team = signalfx_team.prod.id
  dashboard_groups = [signalfx_dashboard_group.infra.id]
}

```

## Argument Reference

The following arguments are supported in the resource block:

* `team` - (Required) ID of team to link to.
* `detectors` - (Optional) List of detector IDs to include in the team.
* `dashboard_groups` - (Optional) List of dashboard group IDs to include in the team.

## Attributes Reference

None.
