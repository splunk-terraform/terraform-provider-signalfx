---
page_title: "Splunk Observability Cloud: signalfx_observability_dashboard"
description: |-
  Manages a Modern Dashboard Template with containers, controls, and layout.
---

# Resource: signalfx_observability_dashboard

This resource manages a complete Dashboard Template document. Its ordered containers can reference existing Template IDs or hold opaque inline JSON. Sections, groups, layout, time range, density, pinned filters, and filter sets are supported. Typed chart blocks will be added separately.

To place the dashboard in a Directory, include `signalfx_observability_dashboard.service.id` in the Directory resource's complete `templates` list. Dashboard updates replace the complete document. Import an existing dashboard and review the generated configuration before applying changes. The provider warns when the stored document contains fields it cannot represent; a later update can remove those fields.

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
    template_id = signalfx_observability_template.chart.id
  }
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

  container {
    layout {
      width  = "6/12"
      height = "2"
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
  }
}
```

## Advanced layout

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

            template_id = signalfx_observability_template.latency.id
          }
        }
      }
    }
  }
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
    template_id = signalfx_observability_template.control_chart.id
  }
}
```

Within each `container`, set exactly one content source. Use `template_id` to reference a reusable Observability Template, or `template_content` for a JSON object containing exactly one dashboard element key. Inline content is preserved without converting unknown chart fields.

Root, section, and group `layout` blocks arrange their immediate containers. A `container.layout` block sets that container's placement. Length arguments accept values such as `"4"`, `"6/12"`, or `"50%"`; use `jsonencode` for clamped lengths or a multi-part `x` or `y` coordinate.

Terraform declaration order controls container order. Controls are written in the order time range, density, pinned filters, then filter set. Complete document updates can remove UI changes that are absent from Terraform configuration.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `title` (String) Dashboard title.

### Optional

- `container` (Block List) Ordered dashboard contents. Terraform declaration order is authoritative. (see [below for nested schema](#nestedblock--container))
- `control_bar` (Block, Optional) Optional controls applied to the dashboard and its charts. (see [below for nested schema](#nestedblock--control_bar))
- `layout` (Block, Optional) Settings for the layout that arranges this level's containers. (see [below for nested schema](#nestedblock--layout))

### Read-Only

- `id` (String) The unique identifier for the resource.

<a id="nestedblock--container"></a>
### Nested Schema for `container`

Optional:

- `group` (Block, Optional) A group containing related containers. (see [below for nested schema](#nestedblock--container--group))
- `layout` (Block, Optional) Placement and size of this container inside its parent layout. Lengths accept numbers or relative strings; clamped values and coordinate arrays can be supplied with jsonencode. (see [below for nested schema](#nestedblock--container--layout))
- `section` (Block, Optional) A section containing containers and optional groups. (see [below for nested schema](#nestedblock--container--section))
- `template_content` (String) Self-contained dashboard JSON object rendered inline in this container.
- `template_id` (String) ID of a reusable Observability Template rendered in this container.

<a id="nestedblock--container--group"></a>
### Nested Schema for `container.group`

Optional:

- `container` (Block List) Ordered contents of this group. (see [below for nested schema](#nestedblock--container--group--container))
- `headerless` (Boolean) Whether to hide the group header.
- `layout` (Block, Optional) Settings for the layout that arranges this level's containers. (see [below for nested schema](#nestedblock--container--group--layout))
- `title` (String) Optional group title. Omit it for an untitled group.

<a id="nestedblock--container--group--container"></a>
### Nested Schema for `container.group.container`

Optional:

- `layout` (Block, Optional) Placement and size of this container inside its parent layout. Lengths accept numbers or relative strings; clamped values and coordinate arrays can be supplied with jsonencode. (see [below for nested schema](#nestedblock--container--group--container--layout))
- `template_content` (String) Self-contained dashboard JSON object rendered inline in this container.
- `template_id` (String) ID of a reusable Observability Template rendered in this container.

<a id="nestedblock--container--group--container--layout"></a>
### Nested Schema for `container.group.container.layout`

Optional:

- `absolute` (Boolean) Whether to position the container independently using its x and y coordinates.
- `height` (String) Starting height of the container.
- `max_height` (String) Maximum height of the container.
- `max_width` (String) Maximum width of the container.
- `min_height` (String) Minimum height of the container.
- `min_width` (String) Minimum width of the container.
- `order` (Number) Display order of the container within its parent layout, overriding declaration order. Reordering in the UI sets this on every container in the layout.
- `width` (String) Starting width of the container.
- `x` (String) Horizontal coordinate. A jsonencoded array is treated as a sum of lengths.
- `y` (String) Vertical coordinate. A jsonencoded array is treated as a sum of lengths.



<a id="nestedblock--container--group--layout"></a>
### Nested Schema for `container.group.layout`

Optional:

- `defaults` (Block, Optional) Default placement and size constraints inherited by every container in this layout. (see [below for nested schema](#nestedblock--container--group--layout--defaults))
- `gap` (Number) Visual gap between adjacent containers in pixels.
- `step` (Number) Layout resolution in pixels. Lengths are rounded to multiples of this value.

<a id="nestedblock--container--group--layout--defaults"></a>
### Nested Schema for `container.group.layout.defaults`

Optional:

- `absolute` (Boolean) Default absolute-positioning behavior.
- `height` (String) Default starting height.
- `max_height` (String) Default maximum height.
- `max_width` (String) Default maximum width.
- `min_height` (String) Default minimum height.
- `min_width` (String) Default minimum width.
- `width` (String) Default starting width.




<a id="nestedblock--container--layout"></a>
### Nested Schema for `container.layout`

Optional:

- `absolute` (Boolean) Whether to position the container independently using its x and y coordinates.
- `height` (String) Starting height of the container.
- `max_height` (String) Maximum height of the container.
- `max_width` (String) Maximum width of the container.
- `min_height` (String) Minimum height of the container.
- `min_width` (String) Minimum width of the container.
- `order` (Number) Display order of the container within its parent layout, overriding declaration order. Reordering in the UI sets this on every container in the layout.
- `width` (String) Starting width of the container.
- `x` (String) Horizontal coordinate. A jsonencoded array is treated as a sum of lengths.
- `y` (String) Vertical coordinate. A jsonencoded array is treated as a sum of lengths.


<a id="nestedblock--container--section"></a>
### Nested Schema for `container.section`

Optional:

- `collapse` (Boolean) Whether the section is currently collapsed.
- `collapsible` (Boolean) Whether the section can be collapsed.
- `container` (Block List) Ordered contents of this section. (see [below for nested schema](#nestedblock--container--section--container))
- `layout` (Block, Optional) Settings for the layout that arranges this level's containers. (see [below for nested schema](#nestedblock--container--section--layout))
- `title` (String) Optional section title. Omit it for an untitled section.

<a id="nestedblock--container--section--container"></a>
### Nested Schema for `container.section.container`

Optional:

- `group` (Block, Optional) A group containing related containers. (see [below for nested schema](#nestedblock--container--section--container--group))
- `layout` (Block, Optional) Placement and size of this container inside its parent layout. Lengths accept numbers or relative strings; clamped values and coordinate arrays can be supplied with jsonencode. (see [below for nested schema](#nestedblock--container--section--container--layout))
- `template_content` (String) Self-contained dashboard JSON object rendered inline in this container.
- `template_id` (String) ID of a reusable Observability Template rendered in this container.

<a id="nestedblock--container--section--container--group"></a>
### Nested Schema for `container.section.container.group`

Optional:

- `container` (Block List) Ordered contents of this group. (see [below for nested schema](#nestedblock--container--section--container--group--container))
- `headerless` (Boolean) Whether to hide the group header.
- `layout` (Block, Optional) Settings for the layout that arranges this level's containers. (see [below for nested schema](#nestedblock--container--section--container--group--layout))
- `title` (String) Optional group title. Omit it for an untitled group.

<a id="nestedblock--container--section--container--group--container"></a>
### Nested Schema for `container.section.container.group.container`

Optional:

- `layout` (Block, Optional) Placement and size of this container inside its parent layout. Lengths accept numbers or relative strings; clamped values and coordinate arrays can be supplied with jsonencode. (see [below for nested schema](#nestedblock--container--section--container--group--container--layout))
- `template_content` (String) Self-contained dashboard JSON object rendered inline in this container.
- `template_id` (String) ID of a reusable Observability Template rendered in this container.

<a id="nestedblock--container--section--container--group--container--layout"></a>
### Nested Schema for `container.section.container.group.container.layout`

Optional:

- `absolute` (Boolean) Whether to position the container independently using its x and y coordinates.
- `height` (String) Starting height of the container.
- `max_height` (String) Maximum height of the container.
- `max_width` (String) Maximum width of the container.
- `min_height` (String) Minimum height of the container.
- `min_width` (String) Minimum width of the container.
- `order` (Number) Display order of the container within its parent layout, overriding declaration order. Reordering in the UI sets this on every container in the layout.
- `width` (String) Starting width of the container.
- `x` (String) Horizontal coordinate. A jsonencoded array is treated as a sum of lengths.
- `y` (String) Vertical coordinate. A jsonencoded array is treated as a sum of lengths.



<a id="nestedblock--container--section--container--group--layout"></a>
### Nested Schema for `container.section.container.group.layout`

Optional:

- `defaults` (Block, Optional) Default placement and size constraints inherited by every container in this layout. (see [below for nested schema](#nestedblock--container--section--container--group--layout--defaults))
- `gap` (Number) Visual gap between adjacent containers in pixels.
- `step` (Number) Layout resolution in pixels. Lengths are rounded to multiples of this value.

<a id="nestedblock--container--section--container--group--layout--defaults"></a>
### Nested Schema for `container.section.container.group.layout.defaults`

Optional:

- `absolute` (Boolean) Default absolute-positioning behavior.
- `height` (String) Default starting height.
- `max_height` (String) Default maximum height.
- `max_width` (String) Default maximum width.
- `min_height` (String) Default minimum height.
- `min_width` (String) Default minimum width.
- `width` (String) Default starting width.




<a id="nestedblock--container--section--container--layout"></a>
### Nested Schema for `container.section.container.layout`

Optional:

- `absolute` (Boolean) Whether to position the container independently using its x and y coordinates.
- `height` (String) Starting height of the container.
- `max_height` (String) Maximum height of the container.
- `max_width` (String) Maximum width of the container.
- `min_height` (String) Minimum height of the container.
- `min_width` (String) Minimum width of the container.
- `order` (Number) Display order of the container within its parent layout, overriding declaration order. Reordering in the UI sets this on every container in the layout.
- `width` (String) Starting width of the container.
- `x` (String) Horizontal coordinate. A jsonencoded array is treated as a sum of lengths.
- `y` (String) Vertical coordinate. A jsonencoded array is treated as a sum of lengths.



<a id="nestedblock--container--section--layout"></a>
### Nested Schema for `container.section.layout`

Optional:

- `defaults` (Block, Optional) Default placement and size constraints inherited by every container in this layout. (see [below for nested schema](#nestedblock--container--section--layout--defaults))
- `gap` (Number) Visual gap between adjacent containers in pixels.
- `step` (Number) Layout resolution in pixels. Lengths are rounded to multiples of this value.

<a id="nestedblock--container--section--layout--defaults"></a>
### Nested Schema for `container.section.layout.defaults`

Optional:

- `absolute` (Boolean) Default absolute-positioning behavior.
- `height` (String) Default starting height.
- `max_height` (String) Default maximum height.
- `max_width` (String) Default maximum width.
- `min_height` (String) Default minimum height.
- `min_width` (String) Default minimum width.
- `width` (String) Default starting width.





<a id="nestedblock--control_bar"></a>
### Nested Schema for `control_bar`

Optional:

- `density` (Block, Optional) The chart density control. Its reserved variable name is always DENSITY. (see [below for nested schema](#nestedblock--control_bar--density))
- `filter_set` (Block, Optional) The ad-hoc filter picker and its optional default filters. Its reserved variable name is always FILTERS. (see [below for nested schema](#nestedblock--control_bar--filter_set))
- `pinned_filter` (Block List) An ordered filter control pinned to the dashboard bar. (see [below for nested schema](#nestedblock--control_bar--pinned_filter))
- `time_range` (Block, Optional) The dashboard time-range control. Its reserved variable name is always TIME. (see [below for nested schema](#nestedblock--control_bar--time_range))

<a id="nestedblock--control_bar--density"></a>
### Nested Schema for `control_bar.density`

Optional:

- `default_variable_value` (Number) Default chart resolution in seconds. Supported values are 30, 60, 120, and 240.
- `description` (String) The density control description.
- `hidden` (Boolean) Whether to hide the density control.
- `label` (String) The density control label.


<a id="nestedblock--control_bar--filter_set"></a>
### Nested Schema for `control_bar.filter_set`

Optional:

- `description` (String) The filter-set control description.
- `filter` (Block List) Ordered default filters for the filter-set control. Empty values are a no-op. (see [below for nested schema](#nestedblock--control_bar--filter_set--filter))
- `hidden` (Boolean) Whether to hide the filter-set control.
- `label` (String) The filter-set control label.

<a id="nestedblock--control_bar--filter_set--filter"></a>
### Nested Schema for `control_bar.filter_set.filter`

Required:

- `key` (String) Property to filter.
- `values` (List of String) Values for the filter. An empty list has no effect.

Optional:

- `disabled` (Boolean) Whether to disable this default filter.
- `match_missing` (Boolean) Whether values missing the property should match.
- `negated` (Boolean) Whether to negate this filter.



<a id="nestedblock--control_bar--pinned_filter"></a>
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


<a id="nestedblock--control_bar--time_range"></a>
### Nested Schema for `control_bar.time_range`

Optional:

- `default_variable_value` (String) Default time-range value, such as -15m, -PT15M, or an absolute time range.
- `description` (String) The time-range control description.
- `hidden` (Boolean) Whether to hide the time-range control.
- `label` (String) The time-range control label.



<a id="nestedblock--layout"></a>
### Nested Schema for `layout`

Optional:

- `defaults` (Block, Optional) Default placement and size constraints inherited by every container in this layout. (see [below for nested schema](#nestedblock--layout--defaults))
- `gap` (Number) Visual gap between adjacent containers in pixels.
- `step` (Number) Layout resolution in pixels. Lengths are rounded to multiples of this value.

<a id="nestedblock--layout--defaults"></a>
### Nested Schema for `layout.defaults`

Optional:

- `absolute` (Boolean) Default absolute-positioning behavior.
- `height` (String) Default starting height.
- `max_height` (String) Default maximum height.
- `max_width` (String) Default maximum width.
- `min_height` (String) Default minimum height.
- `min_width` (String) Default minimum width.
- `width` (String) Default starting width.
