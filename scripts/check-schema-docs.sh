#!/bin/sh

set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
manifest="$root/scripts/migrated-schema-docs.txt"

while read -r kind name; do
	case "$kind" in
	""|'#'*) continue ;;
	resource)
		template="$root/templates/resources/$name.md.tmpl"
		fallback="$root/templates/resources.md.tmpl"
		;;
	data-source)
		template="$root/templates/data-sources/$name.md.tmpl"
		fallback="$root/templates/data-sources.md.tmpl"
		;;
	*)
		echo "Unknown schema documentation kind '$kind' for '$name'" >&2
		exit 1
		;;
	esac

	if [ ! -f "$template" ]; then
		template="$fallback"
	fi

	count=$(rg -c '\{\{[[:space:]]*\.SchemaMarkdown' "$template" || true)
	if [ "$count" -ne 1 ]; then
		echo "$kind '$name' must use exactly one generated SchemaMarkdown section in $template" >&2
		exit 1
	fi

	if rg -n '^#{1,6}[[:space:]]+(Arguments?|Attributes?)([[:space:]]+Reference)?[[:space:]]*$' "$template" >/dev/null; then
		echo "$kind '$name' manually documents schema fields in $template" >&2
		exit 1
	fi
done < "$manifest"
