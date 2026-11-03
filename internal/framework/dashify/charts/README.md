# Schema-powered charts

The `signalfx_observability_chart` resource discovers curated YAML in `definitions/`. Those definitions drive typed Terraform attributes, validation, JSON encoding and imported state. The YAML catalog loads once and returns initialization errors as diagnostics.

To add a chart using existing types and mappings:

1. Copy a similar file from `definitions/` and choose its Terraform name and product element tag.
2. Review the product's chart schema and UI behavior, then author the supported fields, mappings, descriptions and validation rules. Shared definitions live in `definitions/common/`: metric charts extend `metric_chart.yml` (widget presentation, links and a datasource with a nonempty program), Text extends `widget.yml` only, and individual shared fields such as units, precision and timestamp are selected with `ref: common/fields.yml#<field>`. Never `extends` `fields.yml`, which would pull in every field.
3. Extend the six-chart lifecycle fixtures with an independently written expected JSON document when adding a chart. Add focused interpreter tests for any new transformation or field type.
4. Run `go test . ./internal/framework/dashify/...` from the repository root, then `make gen-docs` when the public schema is ready.

No Go registry, native model or generated source changes are needed for charts using existing interpreter capabilities. Runtime writes validate configured values against the curated YAML rules. Reads retain representable server values outside those rules and emit warnings, so drift does not block refresh or deletion. Malformed documents and values that cannot be represented without loss still return errors. Tests cover explicit expected documents, lifecycle, import, computed values, refresh stability and unsupported fields; they do not validate against product JSON Schemas.

The resource and its API codec are `resource_observability_chart.go` and `observability_chart.go`. Runtime loading, attribute construction, validation, codecs and strict JSON parsing live in `interpreter/`. Curated YAML and its embed declaration live in `definitions/`. Chart content encoding is separate from Template API operations so dashboard integration can reuse the interpreter.
