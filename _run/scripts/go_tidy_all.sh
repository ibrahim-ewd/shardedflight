#!/usr/bin/env bash

echo "Global run 'go mod tidy'"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
values_dir="$(dirname "$script_dir")"
root_path="$(dirname "$values_dir")"

EXCLUDE_DIRS=( ".git" "tmp" "_run" )

#############################################################################

set -euo pipefail
cd "$root_path"

PRUNE_EXPR=""
for dir in "${EXCLUDE_DIRS[@]}"; do
    PRUNE_EXPR="$PRUNE_EXPR -path ./$dir -prune -o"
done

eval find . $PRUNE_EXPR -name go.mod -print | while read -r MODFILE; do
    MODDIR=$(dirname "$MODFILE")
    echo "  → $MODDIR"
    go -C "$MODDIR" mod tidy
done
