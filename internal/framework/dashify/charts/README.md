# Schema-powered charts

The production `signalfx_observability_chart` resource discovers curated YAML in `definitions/`. Those definitions drive typed Terraform attributes, validation, JSON encoding and imported state. The YAML catalog loads once and returns initialization errors as diagnostics. Product JSON schemas and their compiler support development tools and tests; they are outside the provider's imports and binary.

To add a chart using existing types and mappings:

1. If its product contract is new, copy the unchanged `O11y*.schema.json` from Olly's `apps/app-modern-dashboards/src/_templates/schemas` into `contracts/`.
2. Run `make scaffold-chart` from the repository root. It scans every vendored product contract and creates a YAML starter in `definitions/` for each missing chart, skipping existing definitions. The name comes from the `o11y:` title, with a `metrics_` prefix when `datasource` is required. Each starter contains the chart identity, a `common/widget.yml` reference, and an empty `properties` map.
3. Select and hand-author the supported Terraform fields, mappings, and descriptions after reviewing the UI. Map the current product fields directly. Replace the placeholder description and draft notice. Adjust the shared widget reference if necessary; metric charts also need the shared datasource with a nonempty program, while Text uses widget chrome without metric plumbing or links.
4. Run `go test . ./internal/framework/dashify/...` from the repository root, then `make gen-docs` when the public schema is ready.

Copy product contracts without reformatting them. Make contract corrections in Olly, then refresh the vendored file here. Git records changes to these files; there is no separate source manifest to update.

No Go registry, native model or generated source changes are needed. The tests check every discovered declaration and encoded representative shape against its product contract and exercise chart lifecycles. A new transformation or field type still needs engine support. Representative contract checks do not exhaust every combination. Runtime writes validate the configured values against the curated YAML rules.

Contract validation runs with ordinary Go tests, including the existing CI test job. To run only the contract tests locally, use `go test ./internal/framework/dashify/charts/contracts`. `make gen-docs` refreshes the published resource documentation when ready to include a new chart. Add a meaningful lifecycle fixture when the chart introduces behavior beyond the existing six-chart coverage.

The resource and its API codec are `resource_observability_chart.go` and `observability_chart.go` in this directory. Runtime loading, attribute construction, validation, codecs and strict JSON parsing live in `interpreter/`. Curated YAML and its embed declaration live in `definitions/`. Vendored product JSON Schemas, development validation and representative configuration checks live in `contracts/`; `contracts/cmd/scaffoldchart` creates YAML starters.

The contract tests use `github.com/santhosh-tekuri/jsonschema/v6` to interpret the vendored product JSON Schemas. Those contracts use references, unions, enums, required fields, array items and restrictions on extra properties. The library independently checks our encoder's output against those rules; JSON parsing alone does not enforce them. It remains a tool/test dependency and is excluded from the provider binary.
