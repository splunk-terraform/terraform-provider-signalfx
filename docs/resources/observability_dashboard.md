---
page_title: "Splunk Observability Cloud: signalfx_observability_dashboard"
description: |-
  Manages an Observability dashboard Template using reusable Template references, raw inline dashboard content, and dashboard controls.
---

# Resource: signalfx_observability_dashboard

The dashboard supports reusable Template references, raw inline dashboard JSON, complete persisted layout options, sections, and groups; typed chart blocks are not supported yet.
Terraform owns the complete dashboard Template document. Place the dashboard in a Directory by including `/v2/template/${signalfx_observability_dashboard.service.id}` in the complete `templates` list of a `signalfx_observability_directory` resource.

## Example

```terraform
terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_template" "chart" {
  title        = "Request rate"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = [{
      "<o11y:SingleValue>" = []
      chart                = {}
      datasource = {
        program = "A = data('requests.count').sum().publish('A')"
      }
      widget = {
        title = "Request rate"
      }
    }]
  })
}

resource "signalfx_observability_dashboard" "service" {
  title = "Service overview"

  container {
    layout {
      width  = "6/12"
      height = "2"
    }
    template {
      template_id = signalfx_observability_template.chart.id
    }
  }
}
```

## Inline content example

```terraform
terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_dashboard" "inline_content" {
  title = "Inline dashboard content"

  container {
    layout {
      width  = "6/12"
      height = "2"
    }

    template {
      content = jsonencode({
        "<o11y:SingleValue>" = []
        chart = {
          color = "blue"
        }
        datasource = {
          program = "A = data('requests.count').sum().publish('A')"
        }
        widget = {
          title = "Request rate"
        }
      })
    }
  }
}
```

## Advanced layout example

```terraform
terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_template" "latency" {
  title        = "Service latency"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = [{
      "<o11y:SingleValue>" = []
      chart                = {}
      datasource = {
        program = "A = data('latency.p99').mean().publish('A')"
      }
      widget = {
        title = "Service latency"
      }
    }]
  })
}

resource "signalfx_observability_dashboard" "advanced_layout" {
  title = "Advanced service layout"

  layout {
    gap  = 8
    step = 8

    defaults {
      min_width  = "4"
      min_height = "2"
    }
  }

  container {
    section {
      title       = "Service health"
      collapse    = false
      collapsible = true

      layout {
        gap  = 4
        step = 4
      }

      container {
        group {
          title      = "Latency details"
          headerless = true

          layout {
            gap  = 0
            step = 1

            defaults {
              height = "20"
            }
          }

          container {
            layout {
              absolute  = true
              width     = jsonencode({ value = "1/2", min = 4, max = "100%" })
              height    = "20"
              min_width = "4"
              max_width = "100%"
              x         = jsonencode(["1/4", 8])
              y         = "12"
            }

            template {
              template_id = signalfx_observability_template.latency.id
            }
          }
        }
      }
    }
  }
}
```

## Dashboard controls example

```terraform
terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_template" "control_chart" {
  title        = "Request rate"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = [{
      "<o11y:SingleValue>" = []
      chart                = {}
      datasource = {
        program = "A = data('requests.count').sum().publish('A')"
      }
      widget = {
        title = "Request rate"
      }
    }]
  })
}

resource "signalfx_observability_dashboard" "controls" {
  title = "Service health"

  control_bar {
    time_range {
      label                  = "Time Range"
      default_variable_value = "-15m"
    }

    density {
      default_variable_value = 60
    }

    pinned_filter {
      variable_name          = "service"
      key                    = "service.name"
      default_variable_value = ["checkout"]
      suggested_values       = ["checkout", "payments"]
      required               = true
      application_mode       = "add"
    }

    filter_set {
      hidden = false

      filter {
        key    = "deployment.environment"
        values = ["prod"]
      }
    }
  }

  container {
    template {
      template_id = signalfx_observability_template.control_chart.id
    }
  }
}
```

## Arguments

* `title` - (Required) Dashboard title.
* `layout` - (Optional) Settings for the root layout that arranges dashboard containers.
  * `gap` - (Optional) Visual gap between adjacent containers in pixels. Must be at least `0`.
  * `step` - (Optional) Layout resolution in pixels. Must be at least `1`.
  * `defaults` - (Optional) Default `absolute`, width, height, and min/max constraints inherited by root containers.
* `container` - (Optional) Ordered dashboard containers. Each container contains exactly one `template`, `section`, or `group` block.
  * `layout` - (Optional) Placement of this container in its parent layout. Supports `absolute`, `width`, `height`, `min_width`, `max_width`, `min_height`, `max_height`, `x`, and `y`.
  * `template` - (Optional) Dashboard panel content. Set exactly one of `template_id`, which imports a reusable Template by ID, or `content`, which accepts a self-contained dashboard JSON object rendered inline.
  * `section` - (Optional) Section with an optional `title`, `collapse`, `collapsible`, an optional child `layout`, and ordered `container` blocks. Omit `title` for an untitled section. Section containers may contain a `template` block or a group.
  * `group` - (Optional) Group with an optional `title`, `headerless`, an optional child `layout`, and ordered `container` blocks. Omit `title` for an untitled group. Groups may be placed directly on a dashboard or inside a section. Group containers contain a `template` block.
* `control_bar` - (Optional) Dashboard controls applied to imported or inline panel content. Only configured controls are stored; the UI may add runtime defaults.
  * `time_range` - (Optional) The dashboard time-range control, always serialized as `TIME`. Its `default_variable_value` accepts values such as `-15m`, `-PT15M`, or an absolute time range.
  * `density` - (Optional) The chart density control, always serialized as `DENSITY`. `default_variable_value` must be `30`, `60`, `120`, or `240`.
  * `pinned_filter` - (Optional, Repeatable) An ordered filter control. `variable_name` is required and must be unique; `key` defaults to `variable_name`. `default_variable_value` and `suggested_values` are optional string lists. `application_mode` accepts `add`, `override`, or `ignore`; `override` only applies when a chart query already filters on the key.
  * `filter_set` - (Optional) The ad-hoc filter picker, always serialized as `FILTERS`. Its optional ordered `filter` blocks contain required `key` and `values`; an empty values list is a no-op. Each filter can also set `negated` or `disabled`.

All control blocks support optional `label`, `description`, and `hidden` fields. Pinned filters also support `only_suggest_preferred_values`, `match_missing`, and `required`. Controls are serialized in the canonical order `TIME`, `DENSITY`, pinned filters, `FILTERS`. Preferred suggestions are stored but are not currently consumed by the rendered filter descriptor. UI saves can add default singleton controls; complete-document ownership means a later Terraform update can remove controls not configured here.

Within a `template` block, `template_id` and `content` are mutually exclusive and one is required. `content` is an opaque JSON escape hatch: it must be an object containing exactly one dashboard element key and is stored as the sole child of the panel. Both a direct element such as `<o11y:SingleValue>` and a standalone-style `<Chart>` wrapper are accepted. Use `template_id` rather than an inline `<$import.*>` element. Formatting and object-key ordering differences in `content` do not produce a Terraform change.

Root, section, and group `layout` blocks configure the layout that arranges their immediate child containers. A `container.layout` block instead configures that single container's placement inside its parent.

All length arguments are strings. Use ordinary values such as `"4"`, `"6/12"`, or `"50%"` for simple lengths. Use `jsonencode({ value = "1/2", min = 4, max = "100%" })` for a clamped length, and use a JSON-encoded array only for a multi-part `x` or `y` coordinate.

Terraform container declaration order is authoritative. Reordering containers in the UI is accepted on refresh but is not retained by the next Terraform update. `group.headerless` controls the group header; a chart's widget header remains part of the referenced Template's `widget.headerless` configuration.

## Attributes

* `id` - The dashboard Template record ID.
