---
page_title: "Splunk Observability Cloud: signalfx_observability_dashboard"
description: |-
  Manages a modern dashboard.
---

# Resource: signalfx_observability_dashboard

Use this resource to manage a modern dashboard's flat panels, controls, and basic panel sizing. Each panel can reference a Template or hold inline JSON.

To place the dashboard in a folder, add its ID to a Directory resource's `templates` list. Updates replace the saved dashboard content, so copy UI changes you want to keep into Terraform before applying.

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

  container = [{
    layout = {
      width  = "50%"
      height = "30"
    }
    template_id = signalfx_observability_template.chart.id
  }]
}
```

## Inline content

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

  container = [{
    layout = {
      width      = "6/12"
      height     = "30"
      min_height = "22"
    }

    template_content = jsonencode({
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
  }]
}
```

## Panel sizing

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

resource "signalfx_observability_dashboard" "service_layout" {
  title = "Service layout"

  container = [{
    template_id = signalfx_observability_template.latency.id
    layout = {
      width      = "1/2"
      height     = "228"
      min_width  = "4"
      min_height = "168"
    }
  }]
}
```

## Dashboard controls

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

  control_bar = {
    time_range = {
      label                  = "Time Range"
      default_variable_value = "-15m"
    }

    density = {
      default_variable_value = 60
    }

    pinned_filter = [{
      variable_name          = "service"
      key                    = "service.name"
      default_variable_value = ["checkout"]
      suggested_values       = ["checkout", "payments"]
      required               = true
      application_mode       = "add"
    }]

    filter_set = {
      hidden = false

      filter = [{
        key    = "deployment.environment"
        values = ["prod"]
      }]
    }
  }

  container = [{
    layout = {
      width      = "6/12"
      height     = "30"
      min_height = "22"
    }
    template_id = signalfx_observability_template.control_chart.id
  }]
}
```

Each `container` represents one panel and needs exactly one content source: `template_id` for a reusable Template or `template_content` for inline JSON.

A panel's `layout` supports `width`, `height`, `min_width`, and `min_height`. Numeric sizes are layout values, not pixels; simple fractions and percentages are also supported. Terraform list order controls panel order. Use a `for` expression to generate panels from a list, for example `container = [for id in var.chart_ids : { template_id = id }]`.

For sections, groups, and advanced layout, use `signalfx_observability_template` with the full `spec`. Importing a nested dashboard into this resource returns an error. A flat dashboard with other unsupported fields can be read, but this resource blocks updates that would discard those fields.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `title` (String) Dashboard title.

### Optional

- `container` (Attributes List) Ordered dashboard panels. Terraform declaration order is authoritative. (see [below for nested schema](#nestedatt--container))
- `control_bar` (Attributes) Optional controls applied to the dashboard and its charts. (see [below for nested schema](#nestedatt--control_bar))

### Read-Only

- `id` (String) The unique identifier for the resource.

<a id="nestedatt--container"></a>
### Nested Schema for `container`

Optional:

- `layout` (Attributes) Basic panel sizing. Numeric sizes are layout values, not pixels. (see [below for nested schema](#nestedatt--container--layout))
- `template_content` (String) Self-contained dashboard JSON object rendered inline in this panel.
- `template_id` (String) ID of a reusable Observability Template rendered in this panel.

<a id="nestedatt--container--layout"></a>
### Nested Schema for `container.layout`

Optional:

- `height` (String) Starting height of the panel.
- `min_height` (String) Minimum height of the panel.
- `min_width` (String) Minimum width of the panel.
- `width` (String) Starting width of the panel.



<a id="nestedatt--control_bar"></a>
### Nested Schema for `control_bar`

Optional:

- `density` (Attributes) The chart density control. Its reserved variable name is always DENSITY. (see [below for nested schema](#nestedatt--control_bar--density))
- `filter_set` (Attributes) The ad-hoc filter picker and its optional default filters. Its reserved variable name is always FILTERS. (see [below for nested schema](#nestedatt--control_bar--filter_set))
- `pinned_filter` (Attributes List) Ordered filter controls pinned to the dashboard bar. (see [below for nested schema](#nestedatt--control_bar--pinned_filter))
- `time_range` (Attributes) The dashboard time-range control. Its reserved variable name is always TIME. (see [below for nested schema](#nestedatt--control_bar--time_range))

<a id="nestedatt--control_bar--density"></a>
### Nested Schema for `control_bar.density`

Optional:

- `default_variable_value` (Number) Default chart resolution in seconds. Supported values are 30, 60, 120, and 240.
- `description` (String) The density control description.
- `hidden` (Boolean) Whether to hide the density control.
- `label` (String) The density control label.


<a id="nestedatt--control_bar--filter_set"></a>
### Nested Schema for `control_bar.filter_set`

Optional:

- `description` (String) The filter-set control description.
- `filter` (Attributes List) Ordered default filters for the filter-set control. Empty values are a no-op. (see [below for nested schema](#nestedatt--control_bar--filter_set--filter))
- `hidden` (Boolean) Whether to hide the filter-set control.
- `label` (String) The filter-set control label.

<a id="nestedatt--control_bar--filter_set--filter"></a>
### Nested Schema for `control_bar.filter_set.filter`

Required:

- `key` (String) Property to filter.
- `values` (List of String) Values for the filter. An empty list has no effect.

Optional:

- `disabled` (Boolean) Whether to disable this default filter.
- `match_missing` (Boolean) Whether values missing the property should match.
- `negated` (Boolean) Whether to negate this filter.



<a id="nestedatt--control_bar--pinned_filter"></a>
### Nested Schema for `control_bar.pinned_filter`

Required:

- `variable_name` (String) Unique control identity. TIME, DENSITY, and FILTERS are reserved.

Optional:

- `application_mode` (String) How the filter is applied: add, override, or ignore.
- `default_variable_value` (List of String) Default selected filter values.
- `description` (String) Control description.
- `hidden` (Boolean) Whether to hide this control.
- `key` (String) Property to filter. Defaults to variable_name.
- `label` (String) Control label.
- `match_missing` (Boolean) Whether values missing the property should match.
- `only_suggest_preferred_values` (Boolean) Whether only preferred suggestions should be offered.
- `required` (Boolean) Whether a visible filter must have a selected value before charts render.
- `suggested_values` (List of String) Preferred filter suggestions.


<a id="nestedatt--control_bar--time_range"></a>
### Nested Schema for `control_bar.time_range`

Optional:

- `default_variable_value` (String) Default time-range value, such as -15m, -PT15M, or an absolute time range.
- `description` (String) The time-range control description.
- `hidden` (Boolean) Whether to hide the time-range control.
- `label` (String) The time-range control label.

## Import

After adding a matching resource block, import an existing Dashboard by its Template ID, not a URL or `/v2/template/...` API reference:

```shell
terraform import signalfx_observability_dashboard.service G47WjI1AABc
```

Terraform can update an imported dashboard when this resource supports everything in its saved content. Otherwise, it blocks the update to avoid losing fields. Use `signalfx_observability_template` with the full `spec` for those dashboards, and review its advanced metadata note first. For a legacy dashboard whose Template has an empty spec, use Save As in the UI and import the new Template ID.
